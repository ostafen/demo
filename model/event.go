package model

type EventType string

const (
	CreateEvent EventType = "create"
	UpdateEvent EventType = "update"
	DeleteEvent EventType = "delete"
)

type Event struct {
	Event EventType `json:"event"`
	Data  *Answer   `json:"data"`
}
