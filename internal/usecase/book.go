package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/X1ag/TravelScheduler/internal/domain"
)

var (
	ErrUserIsNotOwner = errors.New("У вас нет прав для редактирования этой книги")
	ErrPagesOverall = errors.New("Превышено общее количество страниц")
	ErrPagesMustNonZero = errors.New("Количество страниц должно быть больше нуля")
	ErrBookNameEmpty = errors.New("Название книги не может быть пустым")
)

type BookUsecase struct {
	bookRepo domain.BookRepository 
	userRepo domain.UserRepository
	reminderRepo domain.ReminderRepository
}

func NewBookUsecase(bookRepo domain.BookRepository, userRepo domain.UserRepository, reminderRepo domain.ReminderRepository) *BookUsecase {
	return &BookUsecase{
		bookRepo: bookRepo,
		userRepo: userRepo,
		reminderRepo: reminderRepo,
	}
}

func (b *BookUsecase) GetByUserID(ctx context.Context, userID int64) ([]*domain.Book, error) {
	return b.bookRepo.GetByUserID(ctx, userID)
}

func (b *BookUsecase) Create(ctx context.Context, book *domain.Book) error {
	if book.TotalPages <= 0 {
		return ErrPagesMustNonZero
	}
	if book.BookName == "" {
		return ErrBookNameEmpty
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
		return ErrUserIsNotOwner 
	}
	if currentPages > book.TotalPages {
		return ErrPagesOverall
	}
	if currentPages < 0 {
		return ErrPagesMustNonZero
	}
	err = b.bookRepo.UpdateProgress(ctx, bookID, currentPages)
	if err != nil {
		return err
	}
	if currentPages == book.TotalPages {
		_ = b.reminderRepo.Create(ctx, &domain.Reminder{
			UserID: int(userID),
			Message: fmt.Sprintf("Вы закончили книгу %s", book.BookName),
			TriggerAt: time.Now(),
		})
	}

	return nil
}