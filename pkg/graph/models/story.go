package models

import (
	"encoding/json"
)

type View struct {
	Type string          `json:"type"`
	Spec json.RawMessage `json:"spec"`
}

type Story struct {
	Name  string `json:"name"`
	Views []View `json:"views"`
}
