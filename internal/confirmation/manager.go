package confirmation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/logger"
)

type ConfirmationStatus string

const (
	StatusPending    ConfirmationStatus = "pending"
	StatusApproved   ConfirmationStatus = "approved"
	StatusRejected   ConfirmationStatus = "rejected"
	StatusTimeout    ConfirmationStatus = "timeout"
)

type ConfirmationRequest struct {
	ID          string             `json:"id"`
	Type        string             `json:"type"`
	Operation   string             `json:"operation"`
	Details     string             `json:"details"`
	RiskLevel   string             `json:"riskLevel"`
	CreatedAt   time.Time          `json:"createdAt"`
	ExpiresAt   time.Time          `json:"expiresAt"`
	Status      ConfirmationStatus `json:"status"`
	ApprovedBy  string             `json:"approvedBy,omitempty"`
	Channel     string             `json:"channel,omitempty"`
	MessageID   string             `json:"messageId,omitempty"`
}

type ConfirmationManager struct {
	requests  map[string]*ConfirmationRequest
	mu        sync.RWMutex
	log       *logger.Logger
	config    *config.Manager
	notifiers []Notifier
	timeout   time.Duration
}

type Notifier interface {
	Name() string
	SendConfirmation(req *ConfirmationRequest) error
	NotifyResult(req *ConfirmationRequest, approved bool)
}

func NewConfirmationManager(cfg *config.Manager, log *logger.Logger) *ConfirmationManager {
	return &ConfirmationManager{
		requests: make(map[string]*ConfirmationRequest),
		log:      log,
		config:   cfg,
		timeout:  5 * time.Minute,
	}
}

func (m *ConfirmationManager) RegisterNotifier(n Notifier) {
	m.notifiers = append(m.notifiers, n)
}

func (m *ConfirmationManager) RequestConfirmation(ctx context.Context, opType, operation, details, riskLevel string) (bool, error) {
	cfg := m.config.Get()

	if cfg.Tools.UnattendedMode {
		m.log.Info("unattended mode, auto-approving", "operation", operation)
		return true, nil
	}

	for _, allowed := range cfg.Tools.AlwaysAllowDangerous {
		if allowed == operation || allowed == opType {
			m.log.Info("operation in always-allow list", "operation", operation)
			return true, nil
		}
	}

	req := &ConfirmationRequest{
		ID:        generateID(),
		Type:      opType,
		Operation: operation,
		Details:   details,
		RiskLevel: riskLevel,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(m.timeout),
		Status:    StatusPending,
	}

	m.mu.Lock()
	m.requests[req.ID] = req
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.requests, req.ID)
		m.mu.Unlock()
	}()

	for _, n := range m.notifiers {
		if err := n.SendConfirmation(req); err != nil {
			m.log.Error("failed to send confirmation", "notifier", n.Name(), "error", err)
		}
	}

	return m.waitForResponse(ctx, req)
}

func (m *ConfirmationManager) waitForResponse(ctx context.Context, req *ConfirmationRequest) (bool, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
			m.mu.RLock()
			current, ok := m.requests[req.ID]
			m.mu.RUnlock()

			if !ok {
				return false, fmt.Errorf("request not found")
			}

			if time.Now().After(req.ExpiresAt) {
				m.mu.Lock()
				req.Status = StatusTimeout
				m.mu.Unlock()
				return false, fmt.Errorf("confirmation timeout")
			}

			if current.Status != StatusPending {
				return current.Status == StatusApproved, nil
			}
		}
	}
}

func (m *ConfirmationManager) Approve(id, approvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.requests[id]
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}

	req.Status = StatusApproved
	req.ApprovedBy = approvedBy

	m.log.Info("operation approved", "id", id, "operation", req.Operation, "by", approvedBy)

	for _, n := range m.notifiers {
		go n.NotifyResult(req, true)
	}

	return nil
}

func (m *ConfirmationManager) Reject(id, rejectedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.requests[id]
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}

	req.Status = StatusRejected
	req.ApprovedBy = rejectedBy

	m.log.Info("operation rejected", "id", id, "operation", req.Operation, "by", rejectedBy)

	for _, n := range m.notifiers {
		go n.NotifyResult(req, false)
	}

	return nil
}

func (m *ConfirmationManager) GetPending() []*ConfirmationRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*ConfirmationRequest
	for _, req := range m.requests {
		if req.Status == StatusPending {
			pending = append(pending, req)
		}
	}
	return pending
}

func (m *ConfirmationManager) GetRequest(id string) (*ConfirmationRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, ok := m.requests[id]
	if !ok {
		return nil, fmt.Errorf("request not found: %s", id)
	}
	return req, nil
}

func (m *ConfirmationManager) ToJSON() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, _ := json.MarshalIndent(m.requests, "", "  ")
	return string(data)
}

func generateID() string {
	return fmt.Sprintf("conf_%d", time.Now().UnixNano())
}

func IsDangerousOperation(operation string) bool {
	dangerousPatterns := []string{
		"rm -rf",
		"rm -r",
		"rm -f",
		"del /",
		"format",
		"fdisk",
		"mkfs",
		"dd if=",
		"chmod 777",
		"chown -R",
		"> /dev/",
		":(){ :|:& };:",
		"wget | sh",
		"curl | sh",
		"curl | bash",
		"apt-get remove",
		"apt-get purge",
		"yum remove",
		"dnf remove",
		"pacman -R",
		"pip uninstall",
		"npm uninstall",
		"git push --force",
		"git reset --hard",
		"DROP TABLE",
		"DROP DATABASE",
		"TRUNCATE",
		"DELETE FROM",
	}

	for _, pattern := range dangerousPatterns {
		if contains(operation, pattern) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
