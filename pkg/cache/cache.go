package cache

import (
	"database/sql"
	"encoding/json"
	"github.com/rs/zerolog"
	"time"
)

type Cacher interface {
	// Get returns true if get a hit in the cache and are able to deserialize
	// into the provided struct
	Get(key string, into any) bool

	// Set will serialize the provided data and store it in our cache
	Set(key, val any)
}

type Result struct {
	CachedResponse []byte
	LastCached     time.Time
	LastTried      time.Time
}

type Client struct {
	expiresAfter time.Duration
	db           *sql.DB
	log          zerolog.Logger
}

func (c *Client) Get(key string, into any) bool {
	res := &Result{}

	err := c.db.QueryRow(`SELECT response_body, created_at, last_tried_update_at FROM http_cache WHERE endpoint = $1`, key).
		Scan(&res.CachedResponse, &res.LastCached, &res.LastTried)
	if err != nil {
		c.log.Info().Err(err).Msgf("cache miss on: %s", key)
		return false
	}

	if time.Since(res.LastCached) > c.expiresAfter {
		c.log.Info().Msgf("cache expiry on: %s", key)
		return false
	}

	err = json.Unmarshal(res.CachedResponse, into)
	if err != nil {
		c.log.Info().Err(err).Msgf("deserializing cached value: %s", key)
		return false
	}

	return true
}

func (c *Client) Set(key, val any) {
	data, err := json.Marshal(val)
	if err != nil {
		c.log.Info().Err(err).Msgf("serializing value for cache: %s", key)
		return
	}

	_, err = c.db.Exec(`INSERT INTO http_cache (endpoint, response_body, created_at, last_tried_update_at) 
		VALUES ($1, $2, $3, $3) ON CONFLICT (endpoint) DO UPDATE SET response_body = $2, created_at= $3, last_tried_update_at = $3`, key, data, time.Now().UTC())
	if err != nil {
		c.log.Info().Err(err).Msgf("updating cache: %s", key)
		return
	}

	return
}

func New(expiresAfter time.Duration, db *sql.DB, log zerolog.Logger) *Client {
	return &Client{
		expiresAfter: expiresAfter,
		db:           db,
		log:          log,
	}
}
