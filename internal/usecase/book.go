package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/X1ag/TravelScheduler/internal/domain"
)


type BookUsecase struct {
	bookRepo     domain.BookRepository
	userRepo     domain.UserRepository
	reminderRepo domain.ReminderRepository
}

func NewBookUsecase(bookRepo domain.BookRepository, userRepo domain.UserRepository, reminderRepo domain.ReminderRepository) *BookUsecase {
	return &BookUsecase{
		bookRepo:     bookRepo,
		userRepo:     userRepo,
		reminderRepo: reminderRepo,
	}
}

func (b *BookUsecase) GetByUserID(ctx context.Context, userID int64) ([]*domain.Book, error) {
	return b.bookRepo.GetByUserID(ctx, userID)
}

func (b *BookUsecase) Create(ctx context.Context, book *domain.Book) error {
	if book.TotalPages <= 0 {
		return domain.ErrPagesMustNonZero
	}
	if book.BookName == "" {
		return domain.ErrBookNameEmpty
	}

	return b.bookRepo.Create(ctx, book)
}

func (b *BookUsecase) Delete(ctx context.Context, bookID int64) error {
	return b.bookRepo.Delete(ctx, bookID)
}

func (b *BookUsecase) UpdateProgress(ctx context.Context, userID int64, bookID int64, currentPages int) error {
	book, err := b.bookRepo.GetByID(ctx, bookID)
	if err != nil {
		return err
	}

	if book.UserID != userID {
		return domain.ErrUserIsNotOwner
	}
	if currentPages > book.TotalPages {
		return domain.ErrPagesOverall
	}
	if currentPages < 0 {
		return domain.ErrPagesMustNonZero
	}
	err = b.bookRepo.UpdateProgress(ctx, bookID, currentPages)
	if err != nil {
		return err
	}
	if currentPages == book.TotalPages {
		_ = b.reminderRepo.Create(ctx, &domain.Reminder{
			UserID:    userID,
			Message:   fmt.Sprintf("Вы закончили книгу %s", book.BookName),
			TriggerAt: time.Now(),
		})
	}

	return nil
}
