package models

type Owner struct {
	Group            string  `json:"group"`
	TeamkatalogenURL *string `json:"teamkatalogenURL"`
	TeamContact      *string `json:"teamContact"`
	ProductAreaID    *string `json:"productAreaID"`
}
