package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/henrywhitaker3/flow"
)

type Processor interface {
	UserExists(context.Context, *sql.DB, string) (bool, error)
	UserIsOwner(context.Context, *sql.DB, string, string, string) (bool, error)
	DatabaseExists(context.Context, *sql.DB, string, string) (bool, error)
	MakeUserOwner(context.Context, *sql.DB, string, string) error
}

var (
	p *processor
)

func NewProcessor() Processor {
	if p == nil {
		fmt.Println("its nil")
		p = &processor{
			userExists:     flow.NewStore[bool](),
			databaseExists: flow.NewStore[bool](),
			databaseOwned:  flow.NewStore[bool](),
		}
	}
	return p
}

type processor struct {
	userExists     *flow.Store[bool]
	databaseExists *flow.Store[bool]
	databaseOwned  *flow.Store[bool]
}

func (p *processor) UserExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	_, ok := p.userExists.Get(name)
	if ok {
		return true, nil
	}

	row := db.QueryRowContext(ctx, "SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = $1 LIMIT 1", name)
	var tu int
	if err := row.Scan(&tu); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	p.userExists.Put(name, true)

	return true, nil
}

func (p *processor) UserIsOwner(ctx context.Context, db *sql.DB, cluster, user, database string) (bool, error) {
	key := fmt.Sprintf("%s:%s", cluster, database)
	if _, ok := p.databaseOwned.Get(key); ok {
		return true, nil
	}
	row := db.QueryRowContext(ctx, "SELECT datdba::regrole FROM pg_database WHERE datname = $1 LIMIT 1", database)
	var owner string
	if err := row.Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if owner != user {
		return false, nil
	}
	p.databaseOwned.Put(key, true)
	return true, nil
}

func (p *processor) MakeUserOwner(ctx context.Context, db *sql.DB, database, user string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE \"%s\" OWNER TO \"%s\"", database, user))
	return err
}

func (p *processor) DatabaseExists(ctx context.Context, db *sql.DB, cluster string, database string) (bool, error) {
	key := fmt.Sprintf("%s:%s", cluster, database)
	if _, ok := p.databaseExists.Get(key); ok {
		return true, nil
	}
	row := db.QueryRowContext(ctx, "SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1 LIMIT 1", database)
	var du int
	if err := row.Scan(&du); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	p.databaseExists.Put(key, true)
	return true, nil
}
