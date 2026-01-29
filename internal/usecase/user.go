package usecase

import (
	"context"
	"errors"

	"github.com/X1ag/TravelScheduler/internal/domain"
)

var (
	ErrTelegramIDEmpty = errors.New("Telegram ID не может быть пустым")
)

type UserUsecase struct {
	userRepo domain.UserRepository
}

func NewUserUsecase(userRepo domain.UserRepository) *UserUsecase {
	return &UserUsecase{
		userRepo: userRepo,
	}
}

func (u *UserUsecase) Create(ctx context.Context, user *domain.User) error {
	if user.TelegramID == 0 {
		return ErrTelegramIDEmpty
	}
	return u.userRepo.Create(ctx, user)
}

func (u *UserUsecase) GetUserByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	return u.userRepo.GetByTelegramID(ctx, telegramID)
}

func (u *UserUsecase) GetUserByID(ctx context.Context, userID int64) (*domain.User, error) {
	return u.userRepo.GetByID(ctx, userID)
}