package participant

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/qrcode"
	"github.com/google/uuid"
)

// Usecase defines participant business logic operations
type Usecase interface {
	Create(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		input CreateParticipantInput,
	) (*entity.Participant, error)
	BulkCreate(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		input BulkCreateInput,
	) (BulkCreateOutput, error)
	GetByID(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		id uuid.UUID,
	) (*entity.Participant, error)
	List(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		input ListParticipantsInput,
	) (ListParticipantsOutput, error)
	Update(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		id uuid.UUID,
		input UpdateParticipantInput,
	) (*entity.Participant, error)
	Delete(ctx context.Context, userID uuid.UUID, isAdmin bool, id uuid.UUID) error
	GetQRCode(
		ctx context.Context,
		userID uuid.UUID,
		isAdmin bool,
		id uuid.UUID,
		format string,
		size int,
	) (QRCodeOutput, error)
}

var _ Usecase = (*participantUsecase)(nil)

type participantUsecase struct {
	participantRepo repository.ParticipantRepository
	eventRepo       repository.EventRepository
	qrGenerator     *qrcode.Generator
}

// NewUsecase creates a new participant usecase instance
func NewUsecase(
	participantRepo repository.ParticipantRepository,
	eventRepo repository.EventRepository,
	qrGenerator *qrcode.Generator,
) Usecase {
	return &participantUsecase{
		participantRepo: participantRepo,
		eventRepo:       eventRepo,
		qrGenerator:     qrGenerator,
	}
}
