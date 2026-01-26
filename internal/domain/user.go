package domain

import (
	"context"
	"errors"
)

var (
	ErrUniqueViolation = "23505"
	ErrUserAlreadyExists = errors.New("Пользователь с таким telegram id уже существует")
)

type User struct {
				ID 					int64
				TelegramID 	int64
				Name 				string 
				Username 		string
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error 
}