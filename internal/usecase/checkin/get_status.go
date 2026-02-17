package checkin

import (
	"context"
	"errors"
	"fmt"

	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// GetStatus retrieves the check-in status for a participant
func (u *checkinUsecase) GetStatus(
	ctx context.Context,
	userID uuid.UUID,
	isAdmin bool,
	participantID uuid.UUID,
) (*CheckInStatusOutput, error) {
	// Find participant
	participant, err := u.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	// Verify event exists and check authorization
	event, err := u.eventRepo.FindByID(ctx, participant.EventID)
	if err != nil {
		return nil, err
	}

	// Authorization: event owner or admin only
	if !isAdmin && event.OrganizerID != userID {
		return nil, apperrors.Forbidden("you do not have permission to view check-in status for this event")
	}

	// Check if participant has checked in
	checkin, err := u.checkinRepo.FindByParticipant(ctx, participantID)
	if err != nil {
		// If not found, participant hasn't checked in yet
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) && appErr.Code == apperrors.CodeNotFound {
			return &CheckInStatusOutput{
				ParticipantID: participantID,
				IsCheckedIn:   false,
				CheckIn:       nil,
			}, nil
		}
		return nil, fmt.Errorf("failed to get check-in status: %w", err)
	}

	// Build output with check-in details
	output := &CheckInStatusOutput{
		ParticipantID: participantID,
		IsCheckedIn:   true,
		CheckIn: &CheckInOutput{
			ID:               checkin.ID,
			EventID:          checkin.EventID,
			ParticipantID:    participant.ID,
			ParticipantName:  participant.Name,
			ParticipantEmail: participant.Email,
			CheckedInAt:      checkin.CheckedInAt,
			CheckedInBy:      checkin.CheckedInBy,
			Method:           checkin.Method,
		},
	}

	return output, nil
}
