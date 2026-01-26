package domain

import (
	"context"
	"errors"
	"time"
)

type ReminderStatus string

var (
	StatusPending   ReminderStatus = "pending"
	StatusSent      ReminderStatus = "sent"
	StatusFailed    ReminderStatus = "failed"
	StatusCancelled ReminderStatus = "cancelled"
)

var (
	ErrReminderAlreadyExists = errors.New("Уведомление с такими параметрами уже существует")
	ErrReminderNotFound      = errors.New("Уведомление не найдено")
)

type Reminder struct {
	ID        int64     `db:"id"`
	TripId    int64     `db:"trip_id"`
	UserID    int64     `db:"user_id"`
	Message   string    `db:"message"`
	TriggerAt time.Time `db:"trgger_at"`
	Status    string    `db:"status"`
}

type ReminderRepository interface {
	Create(ctx context.Context, reminder *Reminder) error
	GetPending(ctx context.Context, now time.Time) ([]*Reminder, error)
	MarkAsSent(ctx context.Context, id int64) error
}
