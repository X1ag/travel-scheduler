package domain

import (
	"context"
)

type Book struct {
	ID int64
	BookName string 
	Author string
	UserID int64
	TotalPages int 
	CurrentPages int	
}

func (b *Book) Progress() float64 {
	if b.TotalPages == 0 {
		return 0
	}
	return (float64(b.CurrentPages) / float64(b.TotalPages)) * 100
}

type BookRepository interface {
	Create(ctx context.Context, book *Book) error 
	Delete(ctx context.Context, id int) error 
	GetByUserID(ctx context.Context, userID int) (*[]Book, error)
	GetByID(ctx context.Context, bookID int) (Book, error)
	UpdateProgress(ctx context.Context, bookID int64, currentPages int) error
}