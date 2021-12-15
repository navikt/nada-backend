package models

import (
	"time"
)

type Session struct {
	Token   string
	Email   string `json:"email"`
	Name    string `json:"name"`
	Created time.Time
	Expires time.Time
}
