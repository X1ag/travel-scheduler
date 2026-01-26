package domain

import (
	"context"
	"time"
)

type Trip struct {
	ID 						int64  // trip id 
	UserID 				int64 `db:"user_id"`// user id from database, whos traveling 
	From 					string `db:"from_station"`
	To 						string `db:"to_station"`
	BookID 				*int64 `db:"book_id"` 
	DepartureTime time.Time  `db:"departure_time"`
}

type TripRepository interface {
	Create(ctx context.Context, trip *Trip) error 
	GetByUserID(ctx context.Context, userId int) ([]*Trip, error)
}