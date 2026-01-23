package domain

import "context"

type User struct {
				ID 					int64
				TelegramID 	int64
				Name 				string 
				Username 		string
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error 
}