package event

import (
	"context"
	"fmt"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

var _ Usecase = (*eventUsecase)(nil)

type eventUsecase struct {
	eventRepo repository.EventRepository
}

// NewUsecase creates a new instance of Event Usecase.
func NewUsecase(eventRepo repository.EventRepository) Usecase {
	return &eventUsecase{
		eventRepo: eventRepo,
	}
}

func (u *eventUsecase) Create(ctx context.Context, input CreateEventInput) (*entity.Event, error) {
	event := &entity.Event{
		ID:          uuid.New(),
		OrganizerID: input.OrganizerID,
		Name:        input.Name,
		Description: input.Description,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
		Location:    input.Location,
		Timezone:    input.Timezone,
		Status:      input.Status,
	}

	if err := event.Validate(); err != nil {
		return nil, apperrors.Validation(fmt.Sprintf("event validation failed: %v", err))
	}

	if err := u.eventRepo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}

func (u *eventUsecase) GetByID(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
	event, err := u.eventRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (u *eventUsecase) List(ctx context.Context, input ListEventsInput) (ListEventsOutput, error) {
	filter := repository.EventListFilter{
		OrganizerID: input.OrganizerID,
		Status:      input.Status,
		Search:      input.Search,
	}

	offset := (input.Page - 1) * input.PerPage
	limit := input.PerPage

	events, totalCount, err := u.eventRepo.List(ctx, filter, offset, limit)
	if err != nil {
		return ListEventsOutput{}, fmt.Errorf("failed to list events: %w", err)
	}

	return ListEventsOutput{
		Events:     events,
		TotalCount: totalCount,
	}, nil
}

func (u *eventUsecase) Update(
	ctx context.Context,
	id uuid.UUID,
	organizerID uuid.UUID,
	isAdmin bool,
	input UpdateEventInput,
) (*entity.Event, error) {
	event, err := u.eventRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Authorization check
	if !isAdmin && event.OrganizerID != organizerID {
		return nil, apperrors.Forbidden("you do not have permission to update this event")
	}

	if err := u.applyUpdateInput(event, input); err != nil {
		return nil, err
	}

	if err := event.Validate(); err != nil {
		return nil, apperrors.Validation(fmt.Sprintf("event validation failed: %v", err))
	}

	if err := u.eventRepo.Update(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return event, nil
}

func (u *eventUsecase) GetStats(
	ctx context.Context,
	id uuid.UUID,
	organizerID uuid.UUID,
	isAdmin bool,
) (EventStatsOutput, error) {
	event, err := u.eventRepo.FindByID(ctx, id)
	if err != nil {
		return EventStatsOutput{}, err
	}

	// Authorization check
	if !isAdmin && event.OrganizerID != organizerID {
		return EventStatsOutput{}, apperrors.Forbidden("you do not have permission to view stats for this event")
	}

	stats, err := u.eventRepo.GetStats(ctx, id)
	if err != nil {
		return EventStatsOutput{}, fmt.Errorf("failed to get event stats: %w", err)
	}

	var checkinRate float64
	if stats.TotalParticipants > 0 {
		checkinRate = float64(stats.CheckedInCount) / float64(stats.TotalParticipants)
	}

	// Note: byStatus is a placeholder here as the repository doesn't support it yet
	// In a real implementation, you might need another repo method or join.
	return EventStatsOutput{
		EventID:               id,
		TotalParticipants:     stats.TotalParticipants,
		CheckedInParticipants: stats.CheckedInCount,
		CheckinRate:           checkinRate,
		ByStatus:              make(map[string]int64),
	}, nil
}

func (u *eventUsecase) Delete(
	ctx context.Context,
	id uuid.UUID,
	organizerID uuid.UUID,
	isAdmin bool,
) error {
	event, err := u.eventRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if !isAdmin && event.OrganizerID != organizerID {
		return apperrors.Forbidden("you do not have permission to delete this event")
	}

	if event.IsOngoing() {
		return apperrors.Conflict(fmt.Sprintf(
			"cannot delete event with status '%s'. Complete or cancel the event first",
			event.Status,
		))
	}

	if err := u.eventRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

func (u *eventUsecase) applyUpdateInput(event *entity.Event, input UpdateEventInput) error {
	if input.Name != nil {
		event.Name = *input.Name
	}
	if input.Description != nil {
		event.Description = *input.Description
	}
	if input.StartDate != nil {
		event.StartDate = *input.StartDate
	}
	if input.EndDate != nil {
		event.EndDate = input.EndDate
	}
	if input.Location != nil {
		event.Location = *input.Location
	}
	if input.Timezone != nil {
		event.Timezone = *input.Timezone
	}
	if input.Status != nil {
		if err := event.TransitionTo(*input.Status); err != nil {
			return apperrors.BadRequest(fmt.Sprintf("invalid status transition: %v", err))
		}
	}
	return nil
}
