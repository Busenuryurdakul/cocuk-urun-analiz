package agent

import (
	"context"
	"log"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

const maxRecoveryAttempts = 3

type RecoveryWorker struct {
	Service  *Service
	Leases   *LeaseStore
	Interval time.Duration
	stopCh   chan struct{}
}

func NewRecoveryWorker(svc *Service, leases *LeaseStore, interval time.Duration) *RecoveryWorker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &RecoveryWorker{Service: svc, Leases: leases, Interval: interval, stopCh: make(chan struct{})}
}

func (w *RecoveryWorker) Start() {
	go w.loop()
}

func (w *RecoveryWorker) Stop() {
	close(w.stopCh)
}

func (w *RecoveryWorker) loop() {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.scan(context.Background())
		}
	}
}

func (w *RecoveryWorker) ScanOnce(ctx context.Context) {
	w.scan(ctx)
}

func (w *RecoveryWorker) scan(ctx context.Context) {
	if w.Service == nil || w.Service.Runs == nil {
		return
	}
	runs, err := w.Service.Runs.ListStaleRunning(ctx, 50)
	if err != nil {
		log.Printf("recovery list stale runs: %v", err)
		return
	}
	for _, run := range runs {
		if w.Leases == nil {
			w.failStale(ctx, run, "LEASE_STORE_UNAVAILABLE")
			continue
		}
		held, err := w.Leases.Exists(ctx, run.OrganizationID, run.ID)
		if err != nil {
			continue
		}
		if held {
			continue
		}
		if run.RecoveryAttempt >= maxRecoveryAttempts {
			w.failStale(ctx, run, "MAX_RECOVERY_ATTEMPTS")
			continue
		}
		w.failStale(ctx, run, "STALE_LEASE")
	}
}

func (w *RecoveryWorker) failStale(ctx context.Context, run domain.AnalysisRun, reason string) {
	now := time.Now().UTC()
	run.Status = domain.AnalysisStatusFailed
	run.CurrentPhase = domain.AgentPhaseRunFailed
	run.TerminalReason = reason
	run.TerminalError = "orchestrator lease expired"
	run.CompletedAt = &now
	_ = w.Service.Runs.Update(ctx, run.OrganizationID, &run)
	_ = w.Service.RecordOrchestratorEvent(ctx, run.OrganizationID, run.ID, run.TraceID, domain.AgentPhaseRunFailed, "", domain.AgentEventFailed, map[string]string{"reason": reason})
}
