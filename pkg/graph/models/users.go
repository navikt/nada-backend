package models

type UserInfo struct {
	Name   string   `json:"name"`
	Email  string   `json:"email"`
	Groups []*Group `json:"groups"`
}
