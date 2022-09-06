package models

import "github.com/google/uuid"

type ProductArea struct {
	// id is the id of the product area.
	ID uuid.UUID `json:"id"`
	// externalID is the product area external id in teamkatalogen.
	ExternalID string `json:"externalID"`
	// name is the name of the product area.
	Name string `json:"name"`
	// dataproducts is the dataproducts owned by the product area.
	Dataproducts []*Dataproduct `json:"dataproducts"`
	// stories is the stories owned by the product area.
	Stories []*GraphStory `json:"stories"`
}
