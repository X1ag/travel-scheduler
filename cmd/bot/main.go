package main

import (
	"context"
	"log"
	"time"

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
	
	yandexKey := "6f7478e5-151e-436d-b8ba-ace9a4c05375"
	yandexClient := yandex.NewClient(yandexKey)

	tripUC := usecase.NewTripUsecase(tripRepo, reminderRepo, yandexClient)
	// bookUC := usecase.NewBookUsecase(bookRepo, userRepo, reminderRepo)

	log.Println("test")

	from := "s9613483"
	to := "s9612913"
	now := time.Now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	trains, err := tripUC.Search(ctx, from, to, tomorrow)
	if err != nil {
		log.Fatalf("cannot search trains: %v", err)
	}

	if len(trains) == 0 {
		log.Println("no trains found")
		return
	}

	for _, t := range trains {
		log.Printf("- %s | Отправление: %s | Номер: %s\n", 
			t.Title, 
			t.DepartureTime.Format("15:04"), 
			t.TrainNumber)
	}
}