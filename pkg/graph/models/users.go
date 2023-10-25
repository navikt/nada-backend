package models

import "time"

type UserInfo struct {
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Groups          []*Group  `json:"groups"`
	LoginExpiration time.Time `json:"loginExpiration"`
}
