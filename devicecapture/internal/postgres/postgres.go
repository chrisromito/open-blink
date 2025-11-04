package postgres

import (
	"context"
	"devicecapture/internal/postgres/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type DbConf struct {
	Url string
}

type AppDb struct {
	Db *pgxpool.Pool
}

func NewAppDb() *AppDb {
	return &AppDb{}
}

func (a *AppDb) Connect(url string) error {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return err
	}
	a.Db = pool
	log.Info().Msgf("Connecting to %s", url)
	return nil
}

func (a *AppDb) Ping(ctx context.Context) error {
	return a.Db.Ping(ctx)
}

func (a *AppDb) PrintStats() {
	stats := a.Db.Stat()
	log.Debug().Int32("Connection", stats.TotalConns())
	log.Debug().Int64("Acquired", stats.AcquireCount())
	log.Debug().Int32("IdleConnections", stats.IdleConns())
}

func (a *AppDb) GetQueries() *db.Queries {
	return db.New(a.Db)
}

func GetQueries(appDb AppDb) *db.Queries {
	return db.New(appDb.Db)
}

func NewTestAppDb() (*AppDb, error) {
	appDb := NewAppDb()
	err := appDb.Connect("postgres://postgres:postgres@localhost:5432/test_openblink")
	if err != nil {
		return nil, err
	}
	return appDb, nil
}
