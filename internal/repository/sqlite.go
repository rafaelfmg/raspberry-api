package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/example/raspberry-api/internal/domain"
	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("filme não encontrado")

type Repository struct{ db *sql.DB }

func NewInMemory(ctx context.Context) (*Repository, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	r := &Repository{db: db}
	if err := r.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return r, nil
}

func (r *Repository) Close() error { return r.db.Close() }

func (r *Repository) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS movies (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			year      INTEGER NOT NULL,
			title     TEXT    NOT NULL,
			studios   TEXT    NOT NULL,
			producers TEXT    NOT NULL,
			winner    INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_movies_year   ON movies(year)`,
		`CREATE INDEX IF NOT EXISTS idx_movies_winner ON movies(winner)`,
	}
	for _, s := range stmts {
		if _, err := r.db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) BulkInsert(ctx context.Context, movies []domain.Movie) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO movies (year, title, studios, producers, winner) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range movies {
		if _, err := stmt.ExecContext(ctx, m.Year, m.Title, m.Studios, m.Producers, m.Winner); err != nil {
			return fmt.Errorf("inserindo %q: %w", m.Title, err)
		}
	}
	return tx.Commit()
}

func (r *Repository) List(ctx context.Context, f domain.Filter) ([]domain.Movie, error) {
	q := `SELECT id, year, title, studios, producers, winner FROM movies`
	var where []string
	var args []any
	if f.Year != nil {
		where = append(where, "year = ?")
		args = append(args, *f.Year)
	}
	if f.Winner != nil {
		where = append(where, "winner = ?")
		args = append(args, *f.Winner)
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY year, id"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	movies := []domain.Movie{}
	for rows.Next() {
		var m domain.Movie
		if err := rows.Scan(&m.ID, &m.Year, &m.Title, &m.Studios, &m.Producers, &m.Winner); err != nil {
			return nil, err
		}
		movies = append(movies, m)
	}
	return movies, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (domain.Movie, error) {
	var m domain.Movie
	err := r.db.QueryRowContext(ctx,
		`SELECT id, year, title, studios, producers, winner FROM movies WHERE id = ?`, id).
		Scan(&m.ID, &m.Year, &m.Title, &m.Studios, &m.Producers, &m.Winner)
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}
