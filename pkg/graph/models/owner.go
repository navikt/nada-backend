package models

type Owner struct {
	Group            string  `json:"group"`
	AADGroup         *string `json:"aadGroup"`
	TeamkatalogenURL *string `json:"teamkatalogenURL"`
	TeamContact      *string `json:"teamContact"`
	ProductAreaID    *string `json:"productAreaID"`
	TeamID           *string `json:"teamID"`
}
