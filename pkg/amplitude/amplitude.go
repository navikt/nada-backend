package amplitude

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type eventProperties struct {
	SideTittel string `json:"sidetittel"`
	Title      string `json:"title"`
	Type       string `json:"type"`
}

type event struct {
	EventType       string          `json:"event_type"`
	UserID          string          `json:"user_id"`
	Time            int             `json:"time"`
	InsertID        string          `json:"insert_id"`
	EventProperties eventProperties `json:"event_properties"`
}

type amplitudeBody struct {
	APIKey string  `json:"api_key"`
	Events []event `json:"events"`
}

type AmplitudeClient struct {
	log    *logrus.Entry
	apiKey string
}

func New(apiKey string, log *logrus.Entry) *AmplitudeClient {
	return &AmplitudeClient{
		log:    log,
		apiKey: apiKey,
	}
}

func (a *AmplitudeClient) PublishEvent(ctx context.Context, title string) error {
	e := amplitudeBody{
		APIKey: a.apiKey,
		Events: []event{
			{
				EventType: "sidevisning",
				UserID:    "nada-backend",
				Time:      int(time.Now().UnixMilli()),
				InsertID:  uuid.New().String(),
				EventProperties: eventProperties{
					SideTittel: "datafortelling",
					Title:      title,
					Type:       "quarto",
				},
			},
		},
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(e); err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, "https://amplitude.nav.no/collect", buf)
	if err != nil {
		return err
	}
	// Bypass isBot check in amplitude-proxy
	request.Header.Add("User-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36")
	_, err = http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	return nil
}
