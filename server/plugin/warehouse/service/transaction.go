package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// TransactionRunner runs one business command against a transaction-scoped handle.
// Services pass their persistence work through the callback so Excel imports can
// atomically share the same transaction without introducing a repository layer.
type TransactionRunner[T any] interface {
	Transaction(context.Context, func(T) error) error
}

// RunInTransaction keeps transaction orchestration independent from the persistence handle.
func RunInTransaction[T any](ctx context.Context, runner TransactionRunner[T], work func(T) error) error {
	if runner == nil {
		return fmt.Errorf("warehouse transaction runner is required")
	}
	if work == nil {
		return fmt.Errorf("warehouse transaction work is required")
	}
	return runner.Transaction(ctx, work)
}

// GormTransactionRunner adapts GORM transactions for warehouse services.
type GormTransactionRunner struct {
	DB *gorm.DB
}

// Transaction commits work only when its callback succeeds; otherwise GORM rolls it back.
func (r GormTransactionRunner) Transaction(ctx context.Context, work func(*gorm.DB) error) error {
	if r.DB == nil {
		return fmt.Errorf("warehouse transaction database is required")
	}
	return r.DB.WithContext(ctx).Transaction(work)
}
