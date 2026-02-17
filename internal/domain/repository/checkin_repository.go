package repository

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
)

//go:generate mockgen -destination=mocks/mock_checkin_repository.go -package=mocks . CheckinRepository

// CheckinStats represents check-in statistics for an event.
type CheckinStats struct {
	TotalParticipants int64   // Total number of participants registered for the event
	CheckedInCount    int64   // Number of participants who have checked in
	CheckinRate       float64 // Percentage of participants checked in (0.0 - 100.0)
}

// CheckinRepository defines the interface for check-in data persistence operations.
type CheckinRepository interface {
	BaseRepository

	// Create creates a new check-in record with duplicate prevention.
	// Returns ErrCheckinAlreadyExists if the participant has already checked in.
	Create(ctx context.Context, checkin *entity.Checkin) error

	// FindByID finds a check-in by its unique ID.
	// Returns ErrNotFound if the check-in does not exist.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Checkin, error)

	// FindByParticipant finds check-in for a participant.
	// Returns ErrNotFound if the participant has not checked in.
	FindByParticipant(ctx context.Context, participantID uuid.UUID) (*entity.Checkin, error)

	// FindByEvent finds all check-ins for an event with pagination.
	// Returns the check-ins and the total count of check-ins for the event.
	FindByEvent(ctx context.Context, eventID uuid.UUID, limit, offset int) ([]*entity.Checkin, int64, error)

	// GetEventStats gets check-in statistics for an event.
	// Returns stats including total participants, checked-in count, and check-in rate.
	GetEventStats(ctx context.Context, eventID uuid.UUID) (*CheckinStats, error)

	// Delete deletes a check-in (undo check-in operation).
	// Returns ErrNotFound if the check-in does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByParticipant checks if a participant has already checked in to an event.
	ExistsByParticipant(ctx context.Context, eventID, participantID uuid.UUID) (bool, error)
}
