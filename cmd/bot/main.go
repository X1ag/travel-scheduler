package main

import (
	"context"
	"log"
	"os"

	"github.com/X1ag/TravelScheduler/internal/infrastructure/yandex"
	"github.com/X1ag/TravelScheduler/internal/repository/postgres"
	"github.com/X1ag/TravelScheduler/internal/usecase"
	"github.com/X1ag/TravelScheduler/transport/telegram"
	"github.com/go-telegram/bot"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
    if err != nil {
        log.Fatal("Ошибка загрузки .env файла")
    }
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
	userRepo := postgres.NewUserRepository(pool)
	bookRepo := postgres.NewBookRepository(pool)

	yandexKey := os.Getenv("YANDEX_API_KEY") 
	yandexClient := yandex.NewClient(yandexKey)

	tripUC := usecase.NewTripUsecase(tripRepo, reminderRepo, yandexClient)
	bookUC := usecase.NewBookUsecase(bookRepo, userRepo, reminderRepo)
	userUC := usecase.NewUserUsecase(userRepo)

	botWrapped := telegram.NewBot(nil, tripUC, bookUC, userUC)

	opts := []bot.Option{}

	botClient, err := bot.New(os.Getenv("BOT_TOKEN"), opts...)
	if err != nil {
		log.Fatal(err)
	}

	botWrapped.AddClient(botClient)
	botWrapped.RegisterHandlers()
	botWrapped.Start(ctx)
}