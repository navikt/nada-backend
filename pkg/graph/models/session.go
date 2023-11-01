package models

import (
	"time"
)

type Session struct {
	Token       string
	AccessToken string
	Email       string `json:"preferred_username"`
	Name        string `json:"name"`
	Created     time.Time
	Expires     time.Time
}
