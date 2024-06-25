package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"github.com/henrywhitaker3/crunchy-users/internal/k8s"
	"github.com/henrywhitaker3/crunchy-users/internal/logger"
	"github.com/henrywhitaker3/flow"
)

var (
	dbs            = flow.NewStore[*sql.DB]()
	userExists     = flow.NewStore[bool]()
	databaseExists = flow.NewStore[bool]()
	databaseOwned  = flow.NewStore[bool]()
)

func HandleCluster(ctx context.Context, cluster k8s.ClusterResult) error {
	logger := logger.Logger(ctx).With("cluster", cluster.Name, "namespace", cluster.Namespace)
	logger.Info("processing cluster")

	db, err := getDb(ctx, cluster)
	if err != nil {
		logger.Errorw("could not open db connection", "error", err)
		return err
	}

	for _, user := range cluster.Users {
		if err := updatePermissions(ctx, logger, db, cluster, user); err != nil {
			logger.Errorw("could not update permissions", "error", err)
			continue
		}
	}

	return nil
}

func updatePermissions(ctx context.Context, l *zap.SugaredLogger, db *sql.DB, cluster k8s.ClusterResult, user k8s.ClusterUser) error {
	l = l.With("user", user.Name)
	l.Debugw("processing user")

	userExists, err := doesUserExist(ctx, db, user.Name)
	if err != nil {
		l.Errorw("could not determine if user exists", "error", err)
		return err
	}
	if !userExists {
		l.Info("user does not exist, skipping")
		return nil
	}

	return processDatabase(ctx, l, db, cluster, user)
}

func doesUserExist(ctx context.Context, db *sql.DB, name string) (bool, error) {
	_, ok := userExists.Get(name)
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

	userExists.Put(name, true)

	return true, nil
}

func processDatabase(ctx context.Context, l *zap.SugaredLogger, db *sql.DB, cluster k8s.ClusterResult, user k8s.ClusterUser) error {
	for _, dbName := range user.Databases {
		ld := l.With("database", dbName)
		ld.Debug("processing database")
		exists, err := doesDatabaseExist(ctx, db, cluster, dbName)
		if err != nil {
			ld.Errorw("could not determine if database exists", "error", err)
			continue
		}
		if !exists {
			ld.Debug("database does not exist, skipping")
			continue
		}
		owned, err := isUserOwner(ctx, db, cluster, user.Name, dbName)
		if err != nil {
			ld.Errorw("could not determine is user owns the databse", "error", err)
			continue
		}
		if owned {
			ld.Debug("user owns the database, skipping")
			continue
		}
		ld.Info("updating database owner")
		if err := makeUserOwner(ctx, db, user.Name, dbName); err != nil {
			ld.Errorw("could not make user the owner", "error", err)
		}
	}
	return nil
}

func makeUserOwner(ctx context.Context, db *sql.DB, user string, database string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE \"%s\" OWNER TO \"%s\"", database, user))
	return err
}

func isUserOwner(ctx context.Context, db *sql.DB, cluster k8s.ClusterResult, user string, name string) (bool, error) {
	key := fmt.Sprintf("%s:%s", cluster.Key(), name)
	if _, ok := databaseOwned.Get(key); ok {
		return true, nil
	}
	row := db.QueryRowContext(ctx, "SELECT datdba::regrole FROM pg_database WHERE datname = $1 LIMIT 1", name)
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
	databaseOwned.Put(key, true)
	return true, nil
}

func doesDatabaseExist(ctx context.Context, db *sql.DB, cluster k8s.ClusterResult, name string) (bool, error) {
	key := fmt.Sprintf("%s:%s", cluster.Key(), name)
	if _, ok := databaseExists.Get(key); ok {
		return true, nil
	}
	row := db.QueryRowContext(ctx, "SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1 LIMIT 1", name)
	var du int
	if err := row.Scan(&du); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	databaseExists.Put(key, true)
	return true, nil
}

func getDb(ctx context.Context, cluster k8s.ClusterResult) (*sql.DB, error) {
	db, ok := dbs.Get(cluster.Key())
	if !ok {
		conn, err := sql.Open("pgx", cluster.Superuser)
		if err != nil {
			return nil, err
		}
		if err := conn.PingContext(ctx); err != nil {
			return nil, err
		}
		dbs.Put(cluster.Key(), conn)
		db = conn
	}

	return db, nil
}
