package domain

import (
	"context"
	"time"
)

type Trip struct {
	TripID int64 
	UserID int64 // telegram id юзера, который едет 
	From string
	To string 
	BookID int64 // книга, которую читаем
	DepartureTime time.Time // время отъезда
}

type TripRepository interface {
	Create(ctx context.Context, trip *Trip) error 
	GetByUserID(ctx context.Context, userId int) ([]*Trip, error)
}