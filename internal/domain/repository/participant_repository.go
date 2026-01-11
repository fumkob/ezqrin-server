package repository

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
)

//go:generate mockgen -destination=mocks/mock_participant_repository.go -package=mocks . ParticipantRepository

// ParticipantListFilter defines filter options for listing participants.
type ParticipantListFilter struct {
	EventID   uuid.UUID
	Status    *entity.ParticipantStatus
	Search    string // Search by name, email, or employee_id
	PaymentStatus *entity.PaymentStatus
}

// ParticipantRepository defines the interface for participant data persistence operations.
// This interface is larger than typical repository interfaces (>3 methods) due to the CRUD pattern
// combined with specialized query methods (FindByQRCode, ExistsByEmail) needed for check-in operations
// and unique constraint validation. All methods are essential for the event check-in domain.
type ParticipantRepository interface {
	BaseRepository

	// Create creates a new participant in the database.
	// The qr_code field must be unique globally.
	Create(ctx context.Context, participant *entity.Participant) error

	// CreateBulk creates multiple participants in a single operation for better performance.
	// Returns the count of successfully created participants.
	CreateBulk(ctx context.Context, participants []*entity.Participant) (int64, error)

	// FindByID retrieves a participant by its unique ID.
	// Returns ErrNotFound if the participant does not exist.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Participant, error)

	// FindByQRCode retrieves a participant by their QR code for check-in.
	// Returns ErrNotFound if the QR code does not exist.
	FindByQRCode(ctx context.Context, qrCode string) (*entity.Participant, error)

	// FindByEvent retrieves all participants for a specific event.
	// Returns an empty slice if the event has no participants.
	FindByEvent(ctx context.Context, eventID uuid.UUID) ([]*entity.Participant, error)

	// List retrieves a paginated and filtered list of participants.
	// Returns the participants and the total count of participants matching the filter.
	List(ctx context.Context, filter ParticipantListFilter, offset, limit int) ([]*entity.Participant, int64, error)

	// Update updates an existing participant's information.
	// Returns ErrNotFound if the participant does not exist.
	Update(ctx context.Context, participant *entity.Participant) error

	// Delete deletes a participant from the database.
	// Should handle cascading deletion of check-ins.
	// Returns ErrNotFound if the participant does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByEmail checks if a participant with the given email exists for a specific event.
	ExistsByEmail(ctx context.Context, eventID uuid.UUID, email string) (bool, error)

	// GetParticipantStats retrieves statistics about participants for an event.
	GetParticipantStats(ctx context.Context, eventID uuid.UUID) (*ParticipantStats, error)
}

// ParticipantStats represents statistics for participants of an event.
type ParticipantStats struct {
	TotalCount        int64
	ConfirmedCount    int64
	TentativeCount    int64
	CancelledCount    int64
	DeclinedCount     int64
	PaidCount         int64
	UnpaidCount       int64
	TotalPaymentAmount float64
}
