package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// EventStatus represents the status of an event in its lifecycle.
type EventStatus string

const (
	// StatusDraft means the event is being prepared and not yet visible to participants.
	StatusDraft EventStatus = "draft"
	// StatusPublished means the event is active and accepting registrations.
	StatusPublished EventStatus = "published"
	// StatusOngoing means the event is currently happening.
	StatusOngoing EventStatus = "ongoing"
	// StatusCompleted means the event has ended.
	StatusCompleted EventStatus = "completed"
	// StatusCancelled means the event was cancelled.
	StatusCancelled EventStatus = "cancelled"
)

// Validation constants for Event entity
const (
	EventNameMinLength        = 1
	EventNameMaxLength        = 255
	EventDescriptionMaxLength = 5000
	EventLocationMaxLength    = 500
)

// Common validation errors for Event entity
var (
	ErrEventNameRequired       = errors.New("event name is required")
	ErrEventNameTooLong        = errors.New("event name must not exceed 255 characters")
	ErrEventDescriptionTooLong = errors.New("event description must not exceed 5000 characters")
	ErrEventStartDateRequired  = errors.New("event start date is required")
	ErrEventEndDateBeforeStart = errors.New("event end date must be after start date")
	ErrEventLocationTooLong    = errors.New("event location must not exceed 500 characters")
	ErrEventStatusInvalid      = errors.New("invalid event status")
	ErrEventInvalidTransition  = errors.New("invalid event status transition")
)

// Event represents an event created by an organizer.
type Event struct {
	ID          uuid.UUID
	OrganizerID uuid.UUID
	Name        string
	Description string
	StartDate   time.Time
	EndDate     *time.Time
	Location    string
	Timezone    string
	Status      EventStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate validates the Event entity fields.
func (e *Event) Validate() error {
	if e.Name == "" {
		return ErrEventNameRequired
	}
	if len(e.Name) > EventNameMaxLength {
		return ErrEventNameTooLong
	}
	if len(e.Description) > EventDescriptionMaxLength {
		return ErrEventDescriptionTooLong
	}
	if e.StartDate.IsZero() {
		return ErrEventStartDateRequired
	}
	if e.EndDate != nil && e.EndDate.Before(e.StartDate) {
		return ErrEventEndDateBeforeStart
	}
	if len(e.Location) > EventLocationMaxLength {
		return ErrEventLocationTooLong
	}
	if !e.IsValidStatus() {
		return ErrEventStatusInvalid
	}
	return nil
}

// IsValidStatus checks if the event status is valid.
func (e *Event) IsValidStatus() bool {
	switch e.Status {
	case StatusDraft, StatusPublished, StatusOngoing, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the event can transition from its current status to the target status.
// Lifecycle: draft -> published -> ongoing -> completed
// draft -> cancelled, published -> cancelled, ongoing -> cancelled (though spec says ongoing -> completed)
func (e *Event) CanTransitionTo(target EventStatus) bool {
	if e.Status == target {
		return true
	}

	switch e.Status {
	case StatusDraft:
		return target == StatusPublished || target == StatusCancelled
	case StatusPublished:
		return target == StatusOngoing || target == StatusCancelled
	case StatusOngoing:
		return target == StatusCompleted || target == StatusCancelled
	case StatusCompleted, StatusCancelled:
		return false // Terminal statuses
	default:
		return false
	}
}

// TransitionTo updates the event status if the transition is valid.
func (e *Event) TransitionTo(target EventStatus) error {
	if !e.CanTransitionTo(target) {
		return ErrEventInvalidTransition
	}
	e.Status = target
	return nil
}

// IsDraft returns true if the event is in draft status.
func (e *Event) IsDraft() bool {
	return e.Status == StatusDraft
}

// IsPublished returns true if the event is published.
func (e *Event) IsPublished() bool {
	return e.Status == StatusPublished
}

// IsOngoing returns true if the event is currently ongoing.
func (e *Event) IsOngoing() bool {
	return e.Status == StatusOngoing
}

// IsCompleted returns true if the event has been completed.
func (e *Event) IsCompleted() bool {
	return e.Status == StatusCompleted
}

// IsCancelled returns true if the event has been cancelled.
func (e *Event) IsCancelled() bool {
	return e.Status == StatusCancelled
}
