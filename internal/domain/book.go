package domain

import (
	"context"
	"errors"
)

var (
	ErrBookAlreadyExists = errors.New("Книга с такими параметрами уже существует")
	ErrBookNotFound      = errors.New("Книга не найдена")
	ErrUserIsNotOwner    = errors.New("У вас нет прав для редактирования этой книги")
	ErrPagesOverall      = errors.New("Превышено общее количество страниц")
	ErrPagesMustNonZero  = errors.New("Количество страниц должно быть больше нуля")
	ErrBookNameEmpty     = errors.New("Название книги не может быть пустым")
)

type Book struct {
	ID           int64  `db:"id"`
	BookName     string `db:"book_name"`
	Author       string `db:"author"`
	UserID       int64  `db:"user_id"`
	TotalPages   int    `db:"total_pages"`
	CurrentPages int    `db:"current_pages"`
}

func (b *Book) Progress() float64 {
	if b.TotalPages == 0 {
		return 0
	}
	return (float64(b.CurrentPages) / float64(b.TotalPages)) * 100
}

type BookRepository interface {
	Create(ctx context.Context, book *Book) error
	Delete(ctx context.Context, id int64) error
	GetByUserID(ctx context.Context, userID int64) ([]*Book, error)
	GetByID(ctx context.Context, bookID int64) (Book, error)
	UpdateProgress(ctx context.Context, bookID int64, currentPages int) error
}
