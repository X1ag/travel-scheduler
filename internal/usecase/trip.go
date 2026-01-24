package usecase

import (
	"context"
	"errors"

	"github.com/X1ag/TravelScheduler/internal/domain"
)

/// rostov code - 9612913
// taganrog code - 9613483
// api code - 6f7478e5-151e-436d-b8ba-ace9a4c05375

///https://api.rasp.yandex-net.ru/v3.0/search/?apikey=6f7478e5-151e-436d-b8ba-ace9a4c05375&format=json&transport_types=suburban&from=s9613483&to=s9612913&lang=ru_RU&page=1&date=2026-01-23

var (
	ErrDepartureTimeEmpty = errors.New("Время отправления не может быть пустым")
	ErrToPlatformEmpty = errors.New("Платформа назначения не может быть пустой")
)

type TripUsecase struct {
	tripRepo domain.TripRepository
}

func NewTripUsecase(tr domain.TripRepository) *TripUsecase {
	return &TripUsecase{
		tripRepo: tr,
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

func (t *TripUsecase) GetByUserID(ctx context.Context, userID int) ([]*domain.Trip, error) {
	return t.tripRepo.GetByUserID(ctx, userID)
}