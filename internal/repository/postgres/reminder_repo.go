package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/X1ag/TravelScheduler/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReminderRepository struct {
	db *pgxpool.Pool
}

func NewReminderRepository(db *pgxpool.Pool) *ReminderRepository {
	return &ReminderRepository{
		db: db,
	}
}

func (r *ReminderRepository) Create(ctx context.Context, reminder *domain.Reminder) error {
	query := `INSERT INTO reminders (trip_id, user_id, message, trigger_at, status)
						VALUES ($1, $2, $3, $4, $5)
						RETURNING id`
	err := r.db.QueryRow(ctx, query, reminder.TripId, reminder.UserID, reminder.Message, reminder.TriggerAt, domain.StatusPending).Scan(&reminder.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == domain.ErrUniqueViolation {
				return domain.ErrReminderAlreadyExists
			}
		}
		return err
	}

	return nil
}

func (r *ReminderRepository) MarkAsSent(ctx context.Context, id int64) error {
	query := `UPDATE reminders SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, domain.StatusSent, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *ReminderRepository) GetPending(ctx context.Context, now time.Time) ([]*domain.Reminder, error) {
	query := `SELECT (id, trip_id, user_id, message, trigger_at, status) FROM reminders WHERE status = $1 and trigger_at <= $2`
	rows, err := r.db.Query(ctx, query, domain.StatusPending, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pendings := make([]*domain.Reminder, 0, 10)
	for rows.Next() {
		reminder := &domain.Reminder{}
		err := rows.Scan(&reminder.ID, &reminder.TripId, &reminder.UserID, &reminder.Message, &reminder.TriggerAt, &reminder.Status)
		if err != nil {
			return nil, err
		}
		pendings = append(pendings, reminder)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return pendings, nil
}
