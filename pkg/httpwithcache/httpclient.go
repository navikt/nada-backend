package httpwithcache

import (
	"database/sql"
	"encoding/json"
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
	var cachedResponse []byte
	var lastCached time.Time
	var lastTried time.Time

	cacheExpire := 2 * time.Hour
	endpoint := req.URL.String()
	sqlerr := cacheDB.QueryRow(`SELECT response_body, created_at,
	 last_tried_update_at FROM http_cache WHERE endpoint = $1`,
		endpoint).Scan(&cachedResponse, &lastCached, &lastTried)
	if sqlerr == nil && isValidResponse(cachedResponse) {
		if time.Since(lastCached) > cacheExpire && time.Since(lastTried) > cacheExpire {
			reqWithoutContext, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
			if err == nil {
				go updateCache(client, reqWithoutContext)
			} else {
				log.WithError(err).Errorf("Failed to create request")
			}
		}
		return cachedResponse, nil
	} else if !isValidResponse(cachedResponse) {
		log.WithError(sqlerr).Errorf("Cached response for %v is invalid", endpoint)
	} else {
		log.WithError(sqlerr).Errorf("Failed to query database for cached request")
	}
	return updateCache(client, req)
}

func isValidResponse(response []byte) bool {
	var jsonData interface{}
	return len(response) > 0 && json.Unmarshal(response, &jsonData) == nil
}

func updateCache(client *http.Client, req *http.Request) ([]byte, error) {
	endpoint := req.URL.String()
	_, err := cacheDB.Exec(`INSERT INTO http_cache (endpoint, response_body, created_at, last_tried_update_at) 
	VALUES ($1, $2, $3, $3) ON CONFLICT (endpoint) DO UPDATE SET last_tried_update_at = $3`, endpoint, "", time.Now().UTC())
	if err != nil {
		log.WithError(err).Infof("Failed to write to database for %v", req.URL.String())
		return nil, err
	}

	body, statusCode, err := doActualRequest(client, req)
	if err != nil || !isSuccessful(statusCode) {
		log.WithError(err).Errorf("Failed to make request to %v, status code: %v", req.URL.String(), statusCode)
		return nil, err
	}

	if isValidResponse(body) {
		_, err = cacheDB.Exec(`INSERT INTO http_cache (endpoint, response_body, created_at, last_tried_update_at) 
		VALUES ($1, $2, $3, $3) ON CONFLICT (endpoint) DO UPDATE SET response_body = $2, created_at= $3, last_tried_update_at = $3`, endpoint, body, time.Now().UTC())
		if err != nil {
			log.WithError(err).Errorf("Failed to save response to database %v", req.URL.String())
			return body, nil
		}
	}

	return body, nil
}

func isSuccessful(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

func doActualRequest(client *http.Client, req *http.Request) ([]byte, int, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}
