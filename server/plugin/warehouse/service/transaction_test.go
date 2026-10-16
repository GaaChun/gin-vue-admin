package service

import (
	"context"
	"errors"
	"testing"
)

type memoryTransactionStore struct {
	committed []string
}

type memoryTransaction struct {
	values []string
}

func (s *memoryTransactionStore) Transaction(_ context.Context, work func(*memoryTransaction) error) error {
	tx := &memoryTransaction{values: append([]string(nil), s.committed...)}
	if err := work(tx); err != nil {
		return err
	}
	s.committed = tx.values
	return nil
}

func TestRunInTransactionCommitsSuccessfulChanges(t *testing.T) {
	store := &memoryTransactionStore{}
	err := RunInTransaction(context.Background(), store, func(tx *memoryTransaction) error {
		tx.values = append(tx.values, "note", "detail")
		return nil
	})
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	if got, want := len(store.committed), 2; got != want || store.committed[0] != "note" || store.committed[1] != "detail" {
		t.Fatalf("committed state = %#v, want note and detail", store.committed)
	}
}

func TestRunInTransactionRollsBackFailedChanges(t *testing.T) {
	store := &memoryTransactionStore{committed: []string{"existing"}}
	err := RunInTransaction(context.Background(), store, func(tx *memoryTransaction) error {
		tx.values = append(tx.values, "note", "detail")
		return errors.New("detail validation failed")
	})
	if err == nil {
		t.Fatal("failed work must return its error")
	}
	if got, want := len(store.committed), 1; got != want || store.committed[0] != "existing" {
		t.Fatalf("failed transaction leaked prior writes: %#v", store.committed)
	}
}
