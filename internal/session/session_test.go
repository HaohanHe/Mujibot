package session

import (
	"testing"
	"time"

	"github.com/HaohanHe/mujibot/internal/logger"
)

func TestNewManager(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	mgr := NewManager(20, 3600, 100, log)
	defer mgr.Close()

	stats := mgr.GetStats()
	if stats["total_sessions"] != 0 {
		t.Errorf("new manager should have 0 sessions, got: %v", stats["total_sessions"])
	}
	if stats["max_sessions"] != 100 {
		t.Errorf("max_sessions should be 100, got: %v", stats["max_sessions"])
	}
}

func TestGetOrCreate(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	mgr := NewManager(20, 3600, 100, log)
	defer mgr.Close()

	// 创建新会话
	sess1 := mgr.GetOrCreate("user1", "telegram", "default")
	if sess1 == nil {
		t.Fatal("session should not be nil")
	}
	if sess1.UserID != "user1" {
		t.Errorf("user_id should be user1, got: %s", sess1.UserID)
	}

	// 获取已存在的会话
	sess2 := mgr.GetOrCreate("user1", "telegram", "default")
	if sess1.ID != sess2.ID {
		t.Error("should return same session for same user/channel/agent")
	}

	// 创建不同用户的会话
	sess3 := mgr.GetOrCreate("user2", "telegram", "default")
	if sess1.ID == sess3.ID {
		t.Error("should create different session for different user")
	}
}

func TestAddMessage(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	mgr := NewManager(5, 3600, 100, log) // 最多5条消息
	defer mgr.Close()

	sess := mgr.GetOrCreate("user1", "telegram", "default")

	// 添加消息
	mgr.AddMessage(sess, "user", "Hello")
	mgr.AddMessage(sess, "assistant", "Hi there!")

	messages := mgr.GetMessages(sess)
	if len(messages) != 2 {
		t.Errorf("should have 2 messages, got: %d", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("first message role should be user, got: %s", messages[0].Role)
	}

	if messages[0].Content != "Hello" {
		t.Errorf("first message content should be Hello, got: %s", messages[0].Content)
	}
}

func TestMessageLimit(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	mgr := NewManager(3, 3600, 100, log) // 最多3条消息
	defer mgr.Close()

	sess := mgr.GetOrCreate("user1", "telegram", "default")

	// 添加超过限制的消息
	mgr.AddMessage(sess, "user", "msg1")
	mgr.AddMessage(sess, "assistant", "msg2")
	mgr.AddMessage(sess, "user", "msg3")
	mgr.AddMessage(sess, "assistant", "msg4")
	mgr.AddMessage(sess, "user", "msg5")

	messages := mgr.GetMessages(sess)
	if len(messages) != 3 {
		t.Errorf("should have at most 3 messages, got: %d", len(messages))
	}

	// 应该保留最新的消息
	if messages[0].Content != "msg3" {
		t.Errorf("oldest message should be msg3, got: %s", messages[0].Content)
	}
	if messages[2].Content != "msg5" {
		t.Errorf("newest message should be msg5, got: %s", messages[2].Content)
	}
}

func TestClear(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	mgr := NewManager(20, 3600, 100, log)
	defer mgr.Close()

	sess := mgr.GetOrCreate("user1", "telegram", "default")

	mgr.AddMessage(sess, "user", "Hello")
	mgr.AddMessage(sess, "assistant", "Hi!")

	mgr.Clear(sess)

	messages := mgr.GetMessages(sess)
	if len(messages) != 0 {
		t.Errorf("messages should be cleared, got: %d", len(messages))
	}
}

func TestDelete(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	mgr := NewManager(20, 3600, 100, log)
	defer mgr.Close()

	mgr.GetOrCreate("user1", "telegram", "default")

	stats := mgr.GetStats()
	if stats["total_sessions"] != 1 {
		t.Errorf("should have 1 session, got: %v", stats["total_sessions"])
	}

	mgr.Delete("user1", "telegram", "default")

	stats = mgr.GetStats()
	if stats["total_sessions"] != 0 {
		t.Errorf("should have 0 sessions after delete, got: %v", stats["total_sessions"])
	}
}

func TestLRU(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	mgr := NewManager(20, 3600, 3, log) // 最多3个会话
	defer mgr.Close()

	// 创建3个会话
	sess1 := mgr.GetOrCreate("user1", "telegram", "default")
	sess2 := mgr.GetOrCreate("user2", "telegram", "default")
	sess3 := mgr.GetOrCreate("user3", "telegram", "default")

	// 访问sess1使其变为最近使用
	_ = mgr.Get("user1", "telegram", "default")

	// 创建第4个会话，应该淘汰最久未使用的
	sess4 := mgr.GetOrCreate("user4", "telegram", "default")

	// sess2应该被淘汰
	if mgr.Get("user2", "telegram", "default") != nil {
		t.Error("user2 session should be evicted")
	}

	// sess1应该还存在
	if mgr.Get("user1", "telegram", "default") == nil {
		t.Error("user1 session should still exist")
	}

	// 忽略未使用变量警告
	_ = sess1
	_ = sess2
	_ = sess3
	_ = sess4
}

func TestConcurrentAccess(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	mgr := NewManager(20, 3600, 100, log)
	defer mgr.Close()

	// 并发创建会话
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			userID := string(rune('0' + id))
			sess := mgr.GetOrCreate(userID, "telegram", "default")
			mgr.AddMessage(sess, "user", "Hello")
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for goroutines")
		}
	}

	stats := mgr.GetStats()
	if stats["total_sessions"] != 10 {
		t.Errorf("should have 10 sessions, got: %v", stats["total_sessions"])
	}
}
