package store

import (
	"context"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestInsertAndGet(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	data := []byte("hello world")
	id, err := s.Insert(ctx, data)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if id == "" {
		t.Fatal("Insert returned empty id")
	}

	rec, err := s.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(rec.Data) != string(data) {
		t.Errorf("data mismatch: got %q, want %q", rec.Data, data)
	}
	if rec.ID != id {
		t.Errorf("id mismatch: got %q, want %q", rec.ID, id)
	}
}

func TestGetNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Get(ctx, "00000000-0000-0000-0000-000000000000")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestInsertEmptyData(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.Insert(ctx, []byte{})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	rec, err := s.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(rec.Data) != 0 {
		t.Errorf("expected empty data, got %d bytes", len(rec.Data))
	}
}

func TestInsertBinaryData(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	data := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	id, err := s.Insert(ctx, data)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	rec, err := s.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(rec.Data) != len(data) {
		t.Fatalf("data length mismatch: got %d, want %d", len(rec.Data), len(data))
	}
	for i := range data {
		if rec.Data[i] != data[i] {
			t.Errorf("byte %d: got 0x%02X, want 0x%02X", i, rec.Data[i], data[i])
		}
	}
}
