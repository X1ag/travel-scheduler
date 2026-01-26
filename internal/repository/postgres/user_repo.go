package postgres

import (
	"context"
	"errors"

	"github.com/X1ag/TravelScheduler/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUniqueViolation = "23505"
	ErrUserAlreadyExists = errors.New("Пользователь с таким telegram id уже существует")
)

type UserRepository struct {
	db *pgxpool.Pool 
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (u *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (telegram_id, name, username) 
						VALUES ($1, $2, $3)
						RETURNING id`
	err := u.db.QueryRow(ctx, query, user.TelegramID, user.Name, user.Username).Scan(&user.ID)

	if err != nil {
		var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        if pgErr.Code == ErrUniqueViolation { 
            return ErrUserAlreadyExists 
        }
    }
		return err 
	}

	return nil
}