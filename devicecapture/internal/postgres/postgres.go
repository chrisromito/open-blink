package postgres

import (
	"context"
	"devicecapture/internal/postgres/db"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DbConf struct {
	Url string
}

type AppDb struct {
	Config *DbConf
	Db     *pgxpool.Pool
}

func NewAppDb(config *DbConf) *AppDb {
	return &AppDb{
		Config: config,
	}
}

func (a *AppDb) DbConfig() (*pgxpool.Config, error) {
	url := a.Config.Url
	log.Printf("Connecting to %s", url)
	dbConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	dbConfig.MaxConns = int32(4)
	dbConfig.MinConns = int32(0)
	dbConfig.MaxConnLifetime = time.Hour
	dbConfig.MaxConnIdleTime = time.Minute * 30
	dbConfig.HealthCheckPeriod = time.Minute
	dbConfig.ConnConfig.ConnectTimeout = time.Second * 3

	dbConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		log.Println("after connect")
		return nil
	}

	dbConfig.AfterRelease = func(c *pgx.Conn) bool {
		log.Println("After releasing the connection pool to the database")
		return true
	}

	dbConfig.BeforeClose = func(c *pgx.Conn) {
		log.Println("Closed the connection pool to the database")
	}

	return dbConfig, nil
}

func (a *AppDb) Connect(ctx context.Context) (*pgxpool.Pool, error) {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	//conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	dbConfig, err := a.DbConfig()
	if err != nil {
		return nil, err
	}
	pool, err2 := pgxpool.NewWithConfig(ctx, dbConfig)
	if err2 != nil {
		return nil, err2
	}
	a.Db = pool
	connection, err := pool.Acquire(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Release()
	log.Println("Pinging database...")
	err = connection.Ping(context.Background())
	if err != nil {
		log.Println("Error pinging database:", err)
		log.Printf("URL: %s", a.Config.Url)
		return nil, err
	}
	return pool, nil
}

func (a *AppDb) PrintStats() {
	stats := a.Db.Stat()
	log.Println("Connections:", stats.TotalConns())
	log.Println("Acquired Connections:", stats.AcquiredConns())
	log.Println("Acquired Count:", stats.AcquireCount())
	log.Println("Acquired Duration:", stats.AcquireDuration())
	log.Println("Acquired Constructing:", stats.ConstructingConns())
	log.Println("Idle Connections:", stats.IdleConns())
}

func GetQueries(appDb AppDb) *db.Queries {
	return db.New(appDb.Db)
}
