package checkin

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/google/uuid"
)

// Usecase defines check-in business logic operations
type Usecase interface {
	CheckIn(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		input CheckInInput,
	) (*CheckInOutput, error)
	GetStatus(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		participantID uuid.UUID,
	) (*CheckInStatusOutput, error)
	List(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		input ListCheckInsInput,
	) (*ListCheckInsOutput, error)
	Cancel(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		checkinID uuid.UUID,
	) error
}

var _ Usecase = (*checkinUsecase)(nil)

type checkinUsecase struct {
	checkinRepo     repository.CheckinRepository
	participantRepo repository.ParticipantRepository
	eventRepo       repository.EventRepository
	qrHMACSecret    string
}

// NewUsecase creates a new check-in usecase instance
func NewUsecase(
	checkinRepo repository.CheckinRepository,
	participantRepo repository.ParticipantRepository,
	eventRepo repository.EventRepository,
	qrHMACSecret string,
) Usecase {
	return &checkinUsecase{
		checkinRepo:     checkinRepo,
		participantRepo: participantRepo,
		eventRepo:       eventRepo,
		qrHMACSecret:    qrHMACSecret,
	}
}
