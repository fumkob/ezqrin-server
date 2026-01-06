package repository

import (
	"context"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
)

//go:generate mockgen -destination=mocks/mock_event_repository.go -package=mocks . EventRepository

// EventListFilter defines filter options for listing events.
type EventListFilter struct {
	OrganizerID *uuid.UUID
	Status      *entity.EventStatus
	Search      string
	StartDate   *time.Time
	EndDate     *time.Time
}

// EventStats represents basic statistics for an event.
type EventStats struct {
	TotalParticipants int64
	CheckedInCount    int64
}

// EventRepository defines the interface for event data persistence operations.
type EventRepository interface {
	BaseRepository

	// Create creates a new event in the database.
	Create(ctx context.Context, event *entity.Event) error

	// FindByID retrieves an event by its unique ID.
	// Returns ErrNotFound if the event does not exist.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Event, error)

	// List retrieves a paginated and filtered list of events.
	// Returns the events and the total count of events matching the filter.
	List(ctx context.Context, filter EventListFilter, offset, limit int) ([]*entity.Event, int64, error)

	// Update updates an existing event's information.
	// Returns ErrNotFound if the event does not exist.
	Update(ctx context.Context, event *entity.Event) error

	// Delete deletes an event from the database.
	// Should implement cascading deletion of participants and check-ins.
	// Returns ErrNotFound if the event does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetStats retrieves basic statistics for an event.
	GetStats(ctx context.Context, id uuid.UUID) (*EventStats, error)
}
