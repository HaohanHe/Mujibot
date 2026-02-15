package health

import (
	"context"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/HaohanHe/mujibot/internal/logger"
)

const (
	MaxMemoryMB         = 100
	GCTriggerMemoryMB   = 80
	CriticalMemoryMB    = 120
	GCFailureThreshold  = 3
	CheckInterval       = 30 * time.Second
	CooldownPeriod      = 60 * time.Second
)

type MemoryGuard struct {
	log              *logger.Logger
	mu               sync.RWMutex
	consecutiveHigh   int
	lastGC           time.Time
	gcFailures       int
	totalRestarts    int
	emergencyMode    bool
	ctx              context.Context
	cancel           context.CancelFunc
	onCritical       func()
}

func NewMemoryGuard(log *logger.Logger, onCritical func()) *MemoryGuard {
	ctx, cancel := context.WithCancel(context.Background())
	return &MemoryGuard{
		log:        log,
		ctx:        ctx,
		cancel:     cancel,
		onCritical: onCritical,
	}
}

func (g *MemoryGuard) Start() {
	go g.monitorLoop()
}

func (g *MemoryGuard) Stop() {
	g.cancel()
}

func (g *MemoryGuard) monitorLoop() {
	ticker := time.NewTicker(CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			g.check()
		}
	}
}

func (g *MemoryGuard) check() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	heapMB := m.HeapAlloc / 1024 / 1024

	g.mu.Lock()
	defer g.mu.Unlock()

	if heapMB > CriticalMemoryMB {
		g.log.Error("critical memory usage, initiating graceful shutdown",
			"heap_mb", heapMB,
			"sys_mb", m.Sys/1024/1024,
			"goroutines", runtime.NumGoroutine())

		if g.onCritical != nil {
			g.onCritical()
		}
		return
	}

	if heapMB > GCTriggerMemoryMB {
		g.consecutiveHigh++

		if time.Since(g.lastGC) < CooldownPeriod {
			g.log.Debug("gc cooldown, skipping",
				"heap_mb", heapMB,
				"consecutive_high", g.consecutiveHigh)
			return
		}

		g.log.Warn("high memory usage, attempting gc",
			"heap_mb", heapMB,
			"consecutive_high", g.consecutiveHigh)

		beforeMB := heapMB
		runtime.GC()
		debug.FreeOSMemory()
		g.lastGC = time.Now()

		runtime.ReadMemStats(&m)
		afterMB := m.HeapAlloc / 1024 / 1024

		if afterMB >= beforeMB-5 {
			g.gcFailures++
			g.log.Warn("gc ineffective",
				"before_mb", beforeMB,
				"after_mb", afterMB,
				"gc_failures", g.gcFailures)

			if g.gcFailures >= GCFailureThreshold {
				g.log.Error("gc failed multiple times, entering emergency mode")
				g.emergencyMode = true
				g.triggerEmergencyRecovery()
			}
		} else {
			g.gcFailures = 0
			g.log.Info("gc successful",
				"freed_mb", beforeMB-afterMB,
				"current_mb", afterMB)
		}
	} else {
		g.consecutiveHigh = 0
		if g.gcFailures > 0 {
			g.gcFailures--
		}
		g.emergencyMode = false
	}
}

func (g *MemoryGuard) triggerEmergencyRecovery() {
	g.log.Warn("emergency recovery initiated")

	g.log.Info("clearing caches")
	runtime.GC()
	debug.FreeOSMemory()

	g.totalRestarts++
	g.log.Warn("emergency recovery completed",
		"total_restarts", g.totalRestarts)
}

func (g *MemoryGuard) GetStats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"heap_mb":          m.HeapAlloc / 1024 / 1024,
		"sys_mb":           m.Sys / 1024 / 1024,
		"goroutines":       runtime.NumGoroutine(),
		"consecutive_high": g.consecutiveHigh,
		"gc_failures":      g.gcFailures,
		"total_restarts":   g.totalRestarts,
		"emergency_mode":   g.emergencyMode,
	}
}

func (g *MemoryGuard) IsEmergencyMode() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.emergencyMode
}

func (g *MemoryGuard) ForceGC() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.log.Info("manual gc triggered")
	runtime.GC()
	debug.FreeOSMemory()
	g.lastGC = time.Now()
}

func SelfRestart() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}

	args := os.Args[1:]
	env := os.Environ()

	p, err := os.StartProcess(self, args, &os.ProcAttr{
		Env:   env,
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		return err
	}

	p.Release()
	os.Exit(0)
	return nil
}
