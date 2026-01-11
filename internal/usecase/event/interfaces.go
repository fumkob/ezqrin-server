package event

import (
	"context"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
)

// CreateEventInput defines the input for creating a new event.
type CreateEventInput struct {
	OrganizerID uuid.UUID
	Name        string
	Description string
	StartDate   time.Time
	EndDate     *time.Time
	Location    string
	Timezone    string
	Status      entity.EventStatus
}

// UpdateEventInput defines the input for updating an existing event.
type UpdateEventInput struct {
	Name        *string
	Description *string
	StartDate   *time.Time
	EndDate     *time.Time
	Location    *string
	Timezone    *string
	Status      *entity.EventStatus
}

// ListEventsInput defines the input for listing events.
type ListEventsInput struct {
	OrganizerID *uuid.UUID
	Status      *entity.EventStatus
	Search      string
	Page        int
	PerPage     int
	Sort        string
	Order       string
}

// ListEventsOutput defines the output for listing events.
type ListEventsOutput struct {
	Events     []*entity.Event
	TotalCount int64
}

// EventStatsOutput defines the output for event statistics.
type EventStatsOutput struct {
	EventID               uuid.UUID
	TotalParticipants     int64
	CheckedInParticipants int64
	CheckinRate           float64
	ByStatus              map[string]int64
}

// Usecase defines the interface for event-related business logic.
type Usecase interface {
	Create(ctx context.Context, input CreateEventInput) (*entity.Event, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Event, error)
	List(ctx context.Context, input ListEventsInput) (ListEventsOutput, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		organizerID uuid.UUID,
		isAdmin bool,
		input UpdateEventInput,
	) (*entity.Event, error)
	Delete(ctx context.Context, id uuid.UUID, organizerID uuid.UUID, isAdmin bool) error
	GetStats(ctx context.Context, id uuid.UUID, organizerID uuid.UUID, isAdmin bool) (EventStatsOutput, error)
}
