package checkin

import (
	"context"
	"fmt"

	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// List retrieves a paginated list of check-ins for an event
func (u *checkinUsecase) List(
	ctx context.Context,
	userID uuid.UUID,
	isAdmin bool,
	input ListCheckInsInput,
) (*ListCheckInsOutput, error) {
	// Verify event exists and check authorization
	event, err := u.eventRepo.FindByID(ctx, input.EventID)
	if err != nil {
		return nil, err
	}

	// Authorization: event owner or admin only
	if !isAdmin && event.OrganizerID != userID {
		return nil, apperrors.Forbidden("you do not have permission to view check-ins for this event")
	}

	// Calculate offset from page
	offset := (input.Page - 1) * input.PerPage

	// Fetch check-ins from repository
	checkins, totalCount, err := u.checkinRepo.FindByEvent(ctx, input.EventID, input.PerPage, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list check-ins: %w", err)
	}

	// Build output with participant details
	outputs := make([]*CheckInOutput, 0, len(checkins))
	for _, checkin := range checkins {
		// Get participant details
		participant, err := u.participantRepo.FindByID(ctx, checkin.ParticipantID)
		if err != nil {
			// Skip if participant not found (should not happen in normal cases)
			continue
		}

		output := &CheckInOutput{
			ID:               checkin.ID,
			EventID:          checkin.EventID,
			ParticipantID:    participant.ID,
			ParticipantName:  participant.Name,
			ParticipantEmail: participant.Email,
			CheckedInAt:      checkin.CheckedInAt,
			CheckedInBy:      checkin.CheckedInBy,
			Method:           checkin.Method,
		}
		outputs = append(outputs, output)
	}

	return &ListCheckInsOutput{
		CheckIns:   outputs,
		TotalCount: totalCount,
	}, nil
}
