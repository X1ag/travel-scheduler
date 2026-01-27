package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTripAlreadyExists = errors.New("Поездка с такими параметрами уже существует")
	ErrTripNotFound      = errors.New("Поездка не найдена")
)

type Trip struct {
	ID            int64     `db:"id"`      // trip id
	UserID        int64     `db:"user_id"` // user id from database, whos traveling
	From          string    `db:"from_station"`
	To            string    `db:"to_station"`
	BookID        *int64    `db:"book_id"`
	DepartureTime time.Time `db:"departure_time"`
}

type TripRepository interface {
	Create(ctx context.Context, trip *Trip) error
	GetByUserID(ctx context.Context, userId int64) ([]*Trip, error)
}
