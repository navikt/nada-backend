package leaderelection

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
)

func IsLeader() (bool, error) {
	electorPath := os.Getenv("ELECTOR_PATH")
	if electorPath == "" {
		// local development
		return true, nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		return false, err
	}

	resp, err := http.Get("http://" + electorPath)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var electorResponse struct {
		Name string
	}

	if err := json.Unmarshal(bodyBytes, &electorResponse); err != nil {
		return false, err
	}

	return hostname == electorResponse.Name, nil
}
