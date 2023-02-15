package httpwithcache

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

var cacheDB *sql.DB

func SetDatabase(db *sql.DB) {
	cacheDB = db
}

func Do(client *http.Client, req *http.Request) ([]byte, error) {
	return DoWithCacheExpire(client, req, 2*time.Hour)
}

func DoWithCacheExpire(client *http.Client, req *http.Request, cacheExpire time.Duration) ([]byte, error) {
	var cachedResponse []byte
	var lastCached time.Time
	var lastTried time.Time

	endpoint := req.Method + " " + req.URL.String()
	sqlerr := cacheDB.QueryRow(`SELECT response_body, created_at,
	 tried_at FROM http_cache WHERE endpoint = $1`,
		endpoint).Scan(&cachedResponse, &lastCached, &lastTried)
	if sqlerr == nil {
		if time.Since(lastCached) > cacheExpire && time.Since(lastTried) > cacheExpire {
			reqWithoutContext, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
			if err == nil {
				go updateCache(client, reqWithoutContext)
			} else {
				log.WithError(err).Errorf("Failed to create request")
			}
		}
		return cachedResponse, nil
	}

	return updateCache(client, req)
}

func updateCache(client *http.Client, req *http.Request) ([]byte, error) {
	endpoint := req.Method + " " + req.URL.String()
	log.Printf("Update cache for %v", endpoint)
	_, err := cacheDB.Exec(`INSERT INTO http_cache (endpoint, response_body, created_at, tried_at) 
	VALUES ($1, $2, $3, $3) ON CONFLICT (endpoint) DO UPDATE SET tried_at = $3`, endpoint, "", time.Now().UTC())
	if err != nil {
		log.WithError(err).Errorf("Failed to write to database for %v", req.URL.String())
		return nil, err
	}

	body, err := doActualRequest(client, req)
	if err != nil {
		log.WithError(err).Errorf("Failed to make request to %v", req.URL.String())
		return nil, err
	}

	_, err = cacheDB.Exec(`INSERT INTO http_cache (endpoint, response_body, created_at, tried_at) 
		VALUES ($1, $2, $3, $3) ON CONFLICT (endpoint) DO UPDATE SET response_body = $2, created_at= $3, tried_at = $3`, endpoint, body, time.Now().UTC())
	if err != nil {
		log.WithError(err).Errorf("Failed to save response to database %v", req.URL.String())
		return body, nil
	}
	return body, nil
}

func doActualRequest(client *http.Client, req *http.Request) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
