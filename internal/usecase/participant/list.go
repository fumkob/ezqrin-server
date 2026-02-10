package participant

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// List retrieves a paginated list of participants with authorization check
func (u *participantUsecase) List(
	ctx context.Context,
	userID uuid.UUID,
	isAdmin bool,
	input ListParticipantsInput,
) (ListParticipantsOutput, error) {
	// Verify event exists and check authorization
	event, err := u.eventRepo.FindByID(ctx, input.EventID)
	if err != nil {
		return ListParticipantsOutput{}, err
	}

	// Authorization: event owner or admin only
	if !isAdmin && event.OrganizerID != userID {
		return ListParticipantsOutput{}, apperrors.Forbidden(
			"you do not have permission to view participants for this event",
		)
	}

	// Calculate pagination
	offset := (input.Page - 1) * input.PerPage
	limit := input.PerPage

	var participants []*entity.Participant
	var totalCount int64

	// Use Search if there's a search query or status filter
	if input.Search != "" {
		participants, totalCount, err = u.participantRepo.Search(ctx, input.EventID, input.Search, offset, limit)
		if err != nil {
			return ListParticipantsOutput{}, err
		}
	} else {
		// Use FindByEventID for simple listing
		participants, totalCount, err = u.participantRepo.FindByEventID(ctx, input.EventID, offset, limit)
		if err != nil {
			return ListParticipantsOutput{}, err
		}
	}

	// Apply status filter in-memory if needed
	// Note: This is not optimal for large datasets. In production, the repository
	// should support status filtering directly in SQL.
	if input.Status != nil {
		filtered := make([]*entity.Participant, 0, len(participants))
		for _, p := range participants {
			if p.Status == *input.Status {
				filtered = append(filtered, p)
			}
		}
		participants = filtered
		totalCount = int64(len(filtered))
	}

	return ListParticipantsOutput{
		Participants: participants,
		TotalCount:   totalCount,
	}, nil
}
