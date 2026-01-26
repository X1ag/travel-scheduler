package postgres

import (
	"context"
	"errors"

	"github.com/X1ag/TravelScheduler/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrBookAlreadyExists = errors.New("Книга с такими параметрами уже существует")
	ErrBookNotFound = errors.New("Книга не найдена")
)

type BookRepository struct {
	db *pgxpool.Pool 
}

func NewBookRepository(db *pgxpool.Pool) *BookRepository {
	return &BookRepository{
		db: db,
	}
}

func (b *BookRepository) Create(ctx context.Context, book *domain.Book) error {
	query := `INSERT INTO books (user_id, book_name, total_pages, current_pages)
						VALUES ($1, $2, $3, $4)
						RETURNING id`
	err := b.db.QueryRow(ctx, query, &book.UserID, &book.BookName, &book.TotalPages, &book.CurrentPages).Scan(&book.ID)

	if err != nil {
		var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        if pgErr.Code == ErrUniqueViolation { 
            return ErrBookAlreadyExists 
        }
    }
		return err 
	}

	return nil
} 

func (b *BookRepository) GetByUserID(ctx context.Context, userID int) ([]*domain.Book, error) {
	query := `SELECT * FROM books WHERE user_id = $1`
	rows, err := b.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := make([]*domain.Book, 10)
	for rows.Next() {
		book := &domain.Book{}
		err := rows.Scan(&book.ID, &book.UserID, &book.BookName, &book.Author, &book.TotalPages, &book.CurrentPages)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	return books, nil
}

func (b *BookRepository) GetByID(ctx context.Context, bookID int) (*domain.Book, error) {
	query := `SELECT * FROM books WHERE id = $1`
	book := &domain.Book{}
	err := b.db.QueryRow(ctx, query, bookID).Scan(&book.ID, &book.UserID, &book.BookName, &book.Author, &book.TotalPages, &book.CurrentPages)

	if err != nil {
		return nil, err
	}

	return book, nil
}

func (b *BookRepository) UpdateProgress(ctx context.Context, bookID int, currentPages int) error {
	query := `TODO: add update query sql here` 
	rows, err := b.db.Exec(ctx, query, bookID, currentPages)

	if err != nil {
		return err
	}

	if rows.RowsAffected() == 0 {
		return ErrBookNotFound
	}
	
	return nil
}
