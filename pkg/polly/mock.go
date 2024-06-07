package polly

import "context"

type PollyMock struct {
	apiURL string
	url    string
}

func NewMock(apiURL string) *PollyMock {
	url := "https://some.url"

	return &PollyMock{
		apiURL: apiURL,
		url:    url,
	}
}

func (m *PollyMock) SearchPolly(ctx context.Context, q string) ([]*QueryPolly, error) {
	var ret []*QueryPolly

	ret = append(ret, &QueryPolly{
		ExternalID: "28570031-e2b3-4110-8864-41ab279e2e0c",
		Name:       "Behandling",
		URL:        m.url + "/28570031-e2b3-4110-8864-41ab279e2e0c",
	})
	return ret, nil
}
