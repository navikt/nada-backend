package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"regexp"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/navikt/nada-backend/pkg/database/gensql"
	// Pin version of sqlc and goose for cli
	"github.com/pressly/goose/v3"
	"github.com/qustavo/sqlhooks/v2"
	_ "github.com/sqlc-dev/sqlc"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Repo struct {
	Querier            Querier
	db                 *sql.DB
	log                *logrus.Entry
	hooks              *Hooks
	centralDataProject string
}

func (r *Repo) Metrics() []prometheus.Collector {
	return []prometheus.Collector{r.hooks.bucket, r.hooks.errors}
}

type Querier interface {
	gensql.Querier
	WithTx(tx *sql.Tx) *gensql.Queries
}

func New(dbConnDSN string, maxIdleConn, maxOpenConn int, log *logrus.Entry, centralDataProject string) (*Repo, error) {
	hooks := NewHooks()
	sql.Register("psqlhooked", sqlhooks.Wrap(&pq.Driver{}, hooks))

	db, err := sql.Open("psqlhooked", dbConnDSN)
	if err != nil {
		return nil, fmt.Errorf("open sql connection: %w", err)
	}
	db.SetMaxIdleConns(maxIdleConn)
	db.SetMaxOpenConns(maxOpenConn)

	goose.SetBaseFS(embedMigrations)

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, fmt.Errorf("goose up: %w", err)
	}

	return &Repo{
		Querier:            gensql.New(db),
		db:                 db,
		log:                log,
		hooks:              hooks,
		centralDataProject: centralDataProject,
	}, nil
}

// Hooks satisfies the sqlhook.Hooks interface
type Hooks struct {
	bucket *prometheus.HistogramVec
	errors *prometheus.CounterVec
}

func (h *Hooks) OnError(ctx context.Context, err error, query string, args ...interface{}) error {
	h.errors.WithLabelValues(nameFromQuery(query), err.Error()).Inc()
	return nil
}

func NewHooks() *Hooks {
	return &Hooks{
		bucket: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "nada",
			Subsystem: "repo",
			Name:      "query_time",
			Help:      "Query time by name in ms",
			Buckets:   prometheus.ExponentialBuckets(10, 5, 5),
		}, []string{"query"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "nada",
			Subsystem: "repo",
			Name:      "errors",
			Help:      "DB query errors",
		}, []string{"query", "error"}),
	}
}

type ctxKey string

// Before hook will print the query with it's args and return the context with the timestamp
func (h *Hooks) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	return context.WithValue(ctx, ctxKey("begin"), time.Now()), nil
}

// After hook will get the timestamp registered on the Before hook and print the elapsed time
func (h *Hooks) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	begin := ctx.Value(ctxKey("begin")).(time.Time)

	name := nameFromQuery(query)
	h.bucket.WithLabelValues(name).Observe(float64(time.Since(begin).Milliseconds()))

	return ctx, nil
}

var sqlNameReg = regexp.MustCompile(`name:\s*([\w\d]+)`)

func nameFromQuery(q string) string {
	submatch := sqlNameReg.FindStringSubmatch(q)
	if len(submatch) > 1 {
		return submatch[1]
	}
	return "Unknown"
}

func (repo *Repo) GetDB() *sql.DB {
	return repo.db
}
