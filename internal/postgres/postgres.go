package postgres

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/henrywhitaker3/crunchy-users/internal/k8s"
	"github.com/henrywhitaker3/crunchy-users/internal/logger"
	"github.com/henrywhitaker3/flow"
)

var (
	dbs = flow.NewStore[*sql.DB]()
)

func HandleCluster(ctx context.Context, cluster k8s.ClusterResult) error {
	logger := logger.Logger(ctx).With("cluster", cluster.Name, "namespace", cluster.Namespace)
	logger.Info("processing cluster")

	db, err := getDb(ctx, cluster)
	if err != nil {
		logger.Errorw("could not open db connection", "error", err)
		return err
	}
	processor := NewProcessor()

	for _, user := range cluster.Users {
		l := logger.With("cluster", cluster.Name, "namespace", cluster.Namespace, "user", user.Name)
		l.Debug("processing user")
		if exists, err := processor.UserExists(ctx, db, user.Name); err != nil {
			l.Errorw("could not determine is user exists", "error", err)
			continue
		} else if !exists {
			l.Debug("user does not exist, skipping")
			continue
		} else {
			l.Debug("user exists")
		}

		for _, database := range user.Databases {
			ld := l.With("database", database)
			ld.Debug("processing database")
			if exists, err := processor.DatabaseExists(ctx, db, cluster.Key(), database); err != nil {
				ld.Errorw("could not determine if database exists", "error", err)
				continue
			} else if !exists {
				ld.Debug("database does not exist, skipping")
				continue
			} else {
				ld.Debug("database exists")
			}

			if owner, err := processor.UserIsOwner(ctx, db, cluster.Key(), user.Name, database); err != nil {
				ld.Errorw("could not determine if user owns the database", "error", err)
				continue
			} else if !owner {
				ld.Debug("updating database owner")
				if err := processor.MakeUserOwner(ctx, db, database, user.Name); err != nil {
					ld.Errorw("could not update database owner", "error", err)
				}
			} else {
				ld.Debug("user is already owner")
			}
		}
	}

	return nil
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
