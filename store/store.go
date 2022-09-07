package store

import "demo/model"

type EventStore interface {
	Create(a *model.Answer) error
	Update(a *model.Answer) error
	Delete(key string) error
	GetAnswer(key string) (*model.Answer, error)
	GetHistory(key string) (EventIterator, error)
}

type EventIterator interface {
	Next() bool
	Value() *model.Answer
}
