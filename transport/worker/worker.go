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
	tripUC      *usecase.TripUsecase
	bookUC      *usecase.BookUsecase
	userUC      *usecase.UserUsecase
	reminderRepo *postgres.ReminderRepository
	bot         *telegram.Bot
	mu          sync.Mutex
}

func NewWorker(tripUC *usecase.TripUsecase, bookUC *usecase.BookUsecase, userUC *usecase.UserUsecase, reminderRepo *postgres.ReminderRepository, bot *telegram.Bot) *Worker {
	// Настроим формат логов (включая микросекунды и короткое имя файла)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.Printf("INFO: NewWorker created")

	return &Worker{
		bot:          bot,
		tripUC:       tripUC,
		reminderRepo: reminderRepo,
		bookUC:       bookUC,
		userUC:       userUC,
	}
}

func (w *Worker) StartPolling(ctx context.Context, count int) {
	log.Printf("INFO: StartPolling called, starting %d pollers", count)
	for i := 0; i < count; i++ {
		go func(idx int) {
			log.Printf("INFO: starting poller goroutine #%d", idx)
			w.checkPendings(ctx)
			log.Printf("INFO: poller goroutine #%d exited", idx)
		}(i)
	}
}

func (w *Worker) checkPendings(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("INFO: checkPendings received ctx.Done(), exiting")
			return
		case tickTime := <-ticker.C:
			log.Printf("INFO: tick at %s - checking pendings", tickTime.Format(time.RFC3339))

			pendings, err := w.reminderRepo.GetPending(ctx, time.Now())
			if err != nil {
				log.Printf("ERROR: error getting pendings: %v", err)
				// продолжим на следующем тике
				continue
			}

			if len(pendings) == 0 {
				log.Printf("INFO: no pending reminders found")
				continue
			}

			log.Printf("INFO: found %d pending reminders", len(pendings))

			var wg sync.WaitGroup
			sem := make(chan struct{}, 10) // ограничение конкурентности

			for _, pending := range pendings {
				wg.Add(1)
				sem <- struct{}{}
				p := pending // копируем для горутины
				log.Printf("INFO: scheduling handler for reminder id=%d trip_id=%d user_id=%d trigger_at=%s",
					p.ID, p.TripID, p.UserID, p.TriggerAt.String())

				go func(rem *domain.Reminder) {
					defer wg.Done()
					defer func() { <-sem }()
					if err := w.handlePending(ctx, rem); err != nil {
						log.Printf("ERROR: error handling pending id=%d: %v", rem.ID, err)
					} else {
						log.Printf("INFO: successfully handled pending id=%d", rem.ID)
					}
				}(p)
			}
			wg.Wait()
			log.Printf("INFO: finished processing %d pendings", len(pendings))
		}
	}
}

func (w *Worker) handlePending(ctx context.Context, pending *domain.Reminder) error {
	log.Printf("INFO: handlePending start id=%d trip_id=%d user_id=%d", pending.ID, pending.TripID, pending.UserID)

	user, err := w.userUC.GetUserByID(ctx, pending.UserID)
	if err != nil {
		log.Printf("ERROR: error getting user for reminder id=%d user_id=%d: %v", pending.ID, pending.UserID, err)
		return err
	}
	if user == nil {
		log.Printf("ERROR: user not found for reminder id=%d user_id=%d", pending.ID, pending.UserID)
		return nil 
	}
	log.Printf("INFO: found user id=%d telegram_id=%d for reminder id=%d", user.ID, user.TelegramID, pending.ID)

	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	log.Printf("INFO: sending message for reminder id=%d to telegram_id=%d", pending.ID, user.TelegramID)
	w.mu.Lock()
	w.bot.SendMessage(sendCtx, user.TelegramID, pending.Message); 	
	w.mu.Unlock()
	log.Printf("INFO: message sent for reminder id=%d to telegram_id=%d", pending.ID, user.TelegramID)

	if err := w.reminderRepo.MarkAsSent(ctx, pending.ID); err != nil {
		log.Printf("ERROR: error marking reminder id=%d as sent: %v", pending.ID, err)
		return err
	}
	log.Printf("INFO: reminder id=%d marked as sent", pending.ID)

	return nil
}
