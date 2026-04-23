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
			response   BLOB,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		db.Close()
		return nil, err
	}

	// Migrate existing tables that don't have the response column.
	db.Exec("ALTER TABLE verifications ADD COLUMN response BLOB")

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

func (s *Store) SetResponse(ctx context.Context, id string, response []byte) error {
	result, err := s.db.ExecContext(ctx, "UPDATE verifications SET response = ? WHERE id = ?", response, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetResponse(ctx context.Context, id string) ([]byte, error) {
	row := s.db.QueryRowContext(ctx, "SELECT response FROM verifications WHERE id = ?", id)
	var response []byte
	if err := row.Scan(&response); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return response, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
