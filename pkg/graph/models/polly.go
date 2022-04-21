package models

type PollyResult struct {
	// id from polly
	ID string `json:"id"`
	// name from polly
	Name string `json:"name"`
	// url from polly
	URL string `json:"url"`
}

type PollyInput struct {
	// id from polly
	ID string `json:"id"`
	// name from polly
	Name string `json:"name"`
	// url from polly
	URL string `json:"url"`
}
