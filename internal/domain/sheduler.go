package domain

import (
	"context"
	"time"
)

type Reminder struct {
	ID int
	TripId int
	UserID int 
	Message int 
	TriggerAt time.Time
	Status string 
}

type ReminderRepository interface {
	Create(ctx context.Context, reminder *Reminder) error 
	GetPending(ctx context.Context, now time.Time) ([]Reminder, error)
	MarkAsSent(ctx context.Context, id int) error 
}