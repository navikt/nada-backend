package models

type ProductArea struct {
	// id is the id of the product area.
	ID string `json:"id"`
	// name is the name of the product area.
	Name string `json:"name"`
	//areaType is the type of the product area.
	AreaType string `json:"areaType"`
}

type Team struct {
	// id is the team external id in teamkatalogen.
	ID string `json:"id"`
	// name is the name of the team.
	Name string `json:"name"`
	// productAreaID is the id of the product area.
	ProductAreaID string `json:"productAreaID"`
}
