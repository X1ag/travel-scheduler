package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/X1ag/TravelScheduler/internal/domain"
	"github.com/X1ag/TravelScheduler/internal/repository/postgres"
	"github.com/X1ag/TravelScheduler/internal/usecase"
	"github.com/X1ag/TravelScheduler/transport/telegram"
)

type Worker struct {
	tripUC *usecase.TripUsecase
	bookUC *usecase.BookUsecase
	userUC *usecase.UserUsecase
	reminderRepo *postgres.ReminderRepository
	bot *telegram.Bot
	mu sync.Mutex
}

func NewWorker(tripUC *usecase.TripUsecase, bookUC *usecase.BookUsecase, userUC *usecase.UserUsecase, reminderRepo *postgres.ReminderRepository, bot *telegram.Bot) *Worker {
	return &Worker{
		bot: bot,
		tripUC: tripUC,
		reminderRepo: reminderRepo,
		bookUC: bookUC,
		userUC: userUC,
	}
}

func (w *Worker) StartPolling(ctx context.Context, count int) {
	for i := 0; i < count; i++ {
		go w.checkPendings(ctx)
	}
}

func (w *Worker) checkPendings(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			pendings, err := w.reminderRepo.GetPending(ctx, time.Now())
			if err != nil {
				log.Println("error getting pendings", err)	
			}
			var wg sync.WaitGroup
			sem := make(chan struct{}, 10)

			for _, pending := range pendings {
				wg.Add(1)
				sem <- struct{}{}
				p := pending
				go func() {
					defer wg.Done()
					defer func() { <-sem  }()
					if err := w.handlePending(ctx, p); err != nil {
						log.Println("error handling pending", err)
					}
				}()
			}
			wg.Wait()
		}
	}
}

func (w *Worker) handlePending(ctx context.Context, pending *domain.Reminder) error {
	userChatID, err := w.userUC.GetUserByTelegramID(ctx, pending.UserID)
	if err != nil {
		log.Println("error getting user",err)
		return err 
	}
	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	w.mu.Lock()
	w.bot.SendMessage(sendCtx, userChatID.ChatID, pending.Message)
	w.mu.Unlock()

	if err := w.reminderRepo.MarkAsSent(ctx, pending.ID); err != nil {
		log.Println("error marking as sent",err)
		return err
	}
	return nil
}