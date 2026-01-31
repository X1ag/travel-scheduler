package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/X1ag/TravelScheduler/internal/domain"
	"github.com/X1ag/TravelScheduler/internal/utils"
)



var (
	ErrDepartureTimeEmpty = errors.New("Время отправления не может быть пустым")
	ErrToPlatformEmpty = errors.New("Платформа назначения не может быть пустой")
)

type TripUsecase struct {
	tripRepo domain.TripRepository
	reminderRepo domain.ReminderRepository
	yandex domain.ScheduleProvider
}

func NewTripUsecase(tr domain.TripRepository, rr domain.ReminderRepository, yandex domain.ScheduleProvider) *TripUsecase {
	return &TripUsecase{
		tripRepo: tr,
		yandex: yandex,
		reminderRepo: rr,
	}
}

func (t *TripUsecase) Create(ctx context.Context, tr *domain.Trip) error {
	if tr.DepartureTime.IsZero() {
		return ErrDepartureTimeEmpty	
	}
	if tr.To == "" || tr.From == "" {
		return ErrToPlatformEmpty
	}
	return t.tripRepo.Create(ctx, tr)
}

func (t *TripUsecase) GetByUserID(ctx context.Context, userID int64) ([]*domain.Trip, error) {
	return t.tripRepo.GetByUserID(ctx, userID)
}

func (t *TripUsecase) Search(ctx context.Context, from, to string, startDate time.Time) ([]*domain.Schedule, error) {
	allOptions, err := t.yandex.GetNextTrains(ctx, from, to, startDate)
	if err != nil {
		return nil, err
	}

	filteredOptions := t.filteredOptions(allOptions, startDate)

	// retry
	if len(filteredOptions) == 0 {
		tomorrow := time.Date(startDate.Year(), startDate.Month(), startDate.Day()+1, 0, 0, 0, 0, startDate.Location())
		tomorrowOptions, err := t.yandex.GetNextTrains(ctx, from, to, tomorrow)
		if err != nil {
			return nil, err
		}
		filteredOptions = t.filteredOptions(tomorrowOptions, tomorrow)
	}

	// REMOVED: 5-train limit
	// Return ALL available trains, pagination handled in UI layer
	return filteredOptions, nil
}

func (t *TripUsecase) filteredOptions(options []*domain.Schedule, date time.Time) []*domain.Schedule {
	result := make([]*domain.Schedule, 0, 10)
	for _, opt := range options {
		if opt.DepartureTime.After(date) || opt.DepartureTime.Equal(date) {
			result = append(result, opt)
		}
	}
	return result 
}

func (t *TripUsecase) ConfirmTrip(ctx context.Context, tr *domain.Trip) error {
	if err := t.Create(ctx, tr); err != nil {
		return err
	}
	station, exists := utils.GetStationByCode(tr.From)
	if !exists {
		return errors.New("Станция отправления не найдена") 
	}
	return t.reminderRepo.Create(ctx, &domain.Reminder{
		TripID:    tr.ID,
		UserID:    tr.UserID,
		Message:   fmt.Sprintf("Ваша поездка со станции %s начнется через 30 минут! Не опоздайте!", station),
		TriggerAt: tr.DepartureTime.Add(-30 * time.Minute),
		Status: string(domain.StatusPending),
	})
}
