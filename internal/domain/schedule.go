package domain

import (
	"context"
	"time"
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