package repository

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
)

//go:generate mockgen -destination=mocks/mock_participant_repository.go -package=mocks . ParticipantRepository

// ParticipantListFilter defines filter options for listing participants.
type ParticipantListFilter struct {
	EventID *uuid.UUID
	Status  *entity.ParticipantStatus
	Search  string // Search by name, email, or employee_id
}

// ParticipantRepository defines the interface for participant data persistence operations.
type ParticipantRepository interface {
	BaseRepository

	// Create creates a new participant in the database.
	Create(ctx context.Context, participant *entity.Participant) error

	// BulkCreate creates multiple participants in the database with optimized performance.
	BulkCreate(ctx context.Context, participants []*entity.Participant) error

	// FindByID retrieves a participant by its unique ID.
	// Returns ErrNotFound if the participant does not exist.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Participant, error)

	// FindByEventID retrieves paginated participants for an event.
	// Returns the participants and the total count of participants for the event.
	FindByEventID(ctx context.Context, eventID uuid.UUID, offset, limit int) ([]*entity.Participant, int64, error)

	// FindByQRCode retrieves a participant by their QR code.
	// Returns ErrNotFound if no participant has the given QR code.
	FindByQRCode(ctx context.Context, qrCode string) (*entity.Participant, error)

	// Update updates an existing participant's information.
	// Returns ErrNotFound if the participant does not exist.
	Update(ctx context.Context, participant *entity.Participant) error

	// Delete deletes a participant from the database.
	// Returns ErrNotFound if the participant does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// Search searches for participants within an event by name, email, or employee_id.
	// Returns the participants and the total count matching the search criteria.
	Search(
		ctx context.Context,
		eventID uuid.UUID,
		query string,
		offset, limit int,
	) ([]*entity.Participant, int64, error)

	// ExistsByEmail checks if a participant with the given email exists for an event.
	ExistsByEmail(ctx context.Context, eventID uuid.UUID, email string) (bool, error)

	// GetPaymentStats retrieves payment statistics for participants in an event.
	// Used for event deletion validation (Task 7.2).
	GetPaymentStats(ctx context.Context, eventID uuid.UUID) (*ParticipantPaymentStats, error)
}

// ParticipantPaymentStats represents payment statistics for event participants.
type ParticipantPaymentStats struct {
	TotalParticipants  int64
	PaidParticipants   int64
	UnpaidParticipants int64
	TotalPaymentAmount float64
}
