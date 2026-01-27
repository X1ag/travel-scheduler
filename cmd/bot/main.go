package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/X1ag/TravelScheduler/internal/infrastructure/yandex"
	"github.com/X1ag/TravelScheduler/internal/repository/postgres"
	"github.com/X1ag/TravelScheduler/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	dsn := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	if err := postgres.RunMigrations(dsn); err != nil {
		log.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	tripRepo := postgres.NewTripRepository(pool)
	reminderRepo := postgres.NewReminderRepository(pool)
	
	yandexKey := os.Getenv("YANDEX_API_KEY") 
	yandexClient := yandex.NewClient(yandexKey)

	tripUC := usecase.NewTripUsecase(tripRepo, reminderRepo, yandexClient)
	// bookUC := usecase.NewBookUsecase(bookRepo, userRepo, reminderRepo)

	slog.Info("bot started", tripUC)
}