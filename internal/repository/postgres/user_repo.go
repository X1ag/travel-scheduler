package postgres

import (
	"context"
	"errors"
	"log"

	"github.com/X1ag/TravelScheduler/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
			if pgErr.Code == domain.ErrUniqueViolation {
				return domain.ErrUserAlreadyExists
			}
		}
		return err
	}

	return nil
}

func (u *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	query := `SELECT chat_id, id, name, username FROM users WHERE telegram_id = $1`

	user := &domain.User{}
	err := u.db.QueryRow(ctx, query, telegramID).Scan(&user.ID, &user.Name, &user.Username)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return user, nil
}

func (u *UserRepository) GetByID(ctx context.Context, userID int64) (*domain.User, error) {
	query := `SELECT chat_id, id, name, username FROM users WHERE telegram_id = $1`

	user := &domain.User{}
	err := u.db.QueryRow(ctx, query, userID).Scan(&user.ID, &user.ChatID, &user.Name, &user.Username)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return user, nil
}
