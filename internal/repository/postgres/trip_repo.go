package postgres

import (
	"context"
	"errors"

	"github.com/X1ag/TravelScheduler/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTripAlreadyExists = errors.New("Поездка с такими параметрами уже существует")
)

type TripRepository struct {
	db *pgxpool.Pool 
}

func NewTripRepository(db *pgxpool.Pool) *TripRepository {
	return &TripRepository{
		db: db,
	}
}

func (t *TripRepository) Create(ctx context.Context, tr *domain.Trip) error {
	query := `INSERT INTO trips (user_id, from_station, to_station, book_id, departure_time) 
						VALUES ($1, $2, $3, $4, $5)
						RETURNING id`	
	err := t.db.QueryRow(ctx, query, tr.UserID, tr.From, tr.To, tr.BookID, tr.DepartureTime).Scan(&tr.ID)
	if err != nil {
		var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        if pgErr.Code == ErrUniqueViolation { 
            return ErrTripAlreadyExists 
        }
    }
		return err
	}
	return nil
} 

func (t *TripRepository) GetByUserID(ctx context.Context, userID int) ([]*domain.Trip, error) {
	query := `SELECT * FROM trips WHERE user_id = $1`
	rows, err := t.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()	

	trips := make([]*domain.Trip, 10)
	for rows.Next() {
		tr := &domain.Trip{}
		err := rows.Scan(&tr.ID, &tr.UserID, &tr.From, &tr.To, &tr.BookID, &tr.DepartureTime)
		if err != nil {
			return nil, err
		}
		trips = append(trips, tr)
	}	

	return trips, nil 
}