package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"w2w-verification/internal/model"
)

var ErrNotFound = errors.New("record not found")

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS verifications (
			id         TEXT PRIMARY KEY,
			data       BLOB NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Insert(ctx context.Context, data []byte) (string, error) {
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, "INSERT INTO verifications (id, data) VALUES (?, ?)", id, data)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) Get(ctx context.Context, id string) (*model.Record, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, data, created_at FROM verifications WHERE id = ?", id)

	var rec model.Record
	if err := row.Scan(&rec.ID, &rec.Data, &rec.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
