package domain

import (
	"context"
	"errors"
)

var (
	ErrUniqueViolation   = "23505"
	ErrUserAlreadyExists = errors.New("Пользователь с таким telegram id уже существует")
)

type User struct {
	ID         int64 `db:"id"`
	TelegramID int64 `db:"telegram_id"`
	Name       string `db:"name"`
	Username   string `db:"username"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
}
