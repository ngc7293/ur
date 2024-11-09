package ur

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Url struct {
	ID     int
	Slug   string
	Target string
}

type UrlWithHits struct {
	ID     int
	Slug   string
	Target string
	Hits   int
}

type Database interface {
	Insert(slug string, target string) error
	GetUrlFromSlug(slug string) (*Url, error)
	IncrementUrlHitsFromSlug(slug string) error
	ListUrlWithHits() ([]UrlWithHits, error)
}

type SlugConflictError struct{}

func (u *SlugConflictError) Error() string {
	return "Duplicate slug violates unique constraint"
}

type PostgresDatabase struct {
	pool *pgxpool.Pool
}

func NewProgresDatabase(uri string) (*PostgresDatabase, error) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URI"))

	if err != nil {
		return nil, err
	}

	return &PostgresDatabase{pool}, nil
}

func (p *PostgresDatabase) Insert(slug string, target string) error {
	ctx := context.Background()
	conn, err := p.pool.Acquire(ctx)

	if err != nil {
		return err
	}

	_, err = conn.Exec(ctx, "INSERT INTO url(slug,target) VALUES ($1,$2)", slug, target)

	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		return &SlugConflictError{}
	}

	return err
}

func (p *PostgresDatabase) GetUrlFromSlug(slug string) (*Url, error) {
	ctx := context.Background()
	conn, err := p.pool.Acquire(ctx)

	if err != nil {
		return nil, err
	}

	defer conn.Release()

	rows, err := conn.Query(ctx, "SELECT id, target FROM url WHERE slug = $1 LIMIT 1", slug)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var id int
	var target string

	if err = rows.Scan(&id, &target); err != nil {
		return nil, err
	}

	return &Url{id, slug, target}, nil
}

func (p *PostgresDatabase) IncrementUrlHitsFromSlug(slug string) error {
	ctx := context.Background()
	conn, err := p.pool.Acquire(ctx)

	if err != nil {
		return err
	}

	defer conn.Release()

	_, err = conn.Exec(ctx, "UPDATE url SET hits = hits + 1 WHERE slug = $1", slug)
	return err
}

func (p *PostgresDatabase) ListUrlWithHits() ([]UrlWithHits, error) {
	ctx := context.Background()
	conn, err := p.pool.Acquire(ctx)

	if err != nil {
		return []UrlWithHits{}, nil
	}

	defer conn.Release()

	rows, err := conn.Query(ctx, "SELECT id, slug, target, hits FROM url")

	if err != nil {
		return []UrlWithHits{}, nil
	}

	results := []UrlWithHits{}

	for rows.Next() {
		var id int
		var slug string
		var target string
		var hits int

		if err = rows.Scan(&id, &slug, &target, &hits); err != nil {
			return []UrlWithHits{}, nil
		}

		results = append(results, UrlWithHits{id, slug, target, hits})
	}

	return results, nil
}
