package model

type Answer struct {
	Key   string `json:"key" validate:"required"`
	Value string `json:"value" validate:"required"`
}
