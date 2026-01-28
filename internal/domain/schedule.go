package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidInput = errors.New("Неправильный ввод\\. Формат ввода: <откуда\\> <куда\\> <дата\\> <время в формате 15:36:01\\>")
)

type Schedule struct {
	TrainNumber string
	Title string 
	DepartureTime time.Time
	ArrivalTime time.Time
}

type ScheduleProvider interface {
	GetNextTrains(ctx context.Context, fromCode, toCode string, date time.Time) ([]*Schedule, error)
}