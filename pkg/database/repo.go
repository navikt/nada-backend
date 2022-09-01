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
	"github.com/navikt/nada-backend/pkg/event"

	// Pin version of sqlc and goose for cli
	_ "github.com/kyleconroy/sqlc"
	"github.com/pressly/goose/v3"
	"github.com/qustavo/sqlhooks/v2"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Repo struct {
	querier Querier
	db      *sql.DB
	log     *logrus.Entry

	events *event.Manager

	hooks *Hooks
}

func (r *Repo) Metrics() prometheus.Collector {
	return r.hooks.bucket
}

type Querier interface {
	gensql.Querier
	WithTx(tx *sql.Tx) *gensql.Queries
}

func New(dbConnDSN string, maxIdleConn, maxOpenConn int, eventMgr *event.Manager, log *logrus.Entry) (*Repo, error) {
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
		querier: gensql.New(db),
		db:      db,
		log:     log,
		events:  eventMgr,
		hooks:   hooks,
	}, nil
}

// Hooks satisfies the sqlhook.Hooks interface
type Hooks struct {
	bucket *prometheus.HistogramVec
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
