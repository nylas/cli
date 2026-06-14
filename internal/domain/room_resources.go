package domain

// RoomResource represents a bookable room or equipment resource returned by the
// Nylas room resources endpoint (GET /v3/grants/{id}/resources).
//
// A resource's Email doubles as a calendar ID, so it can be added as a
// participant when creating events or passed to availability/free-busy checks.
type RoomResource struct {
	Email        string `json:"email"`
	Name         string `json:"name,omitempty"`
	Capacity     int    `json:"capacity,omitempty"`
	Building     string `json:"building,omitempty"`
	FloorName    string `json:"floor_name,omitempty"`
	FloorSection string `json:"floor_section,omitempty"`
	FloorNumber  int    `json:"floor_number,omitempty"`
	Object       string `json:"object,omitempty"`
}
