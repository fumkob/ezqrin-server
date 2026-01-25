package participant

import (
	"context"

	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// Delete deletes a participant with authorization check
func (u *participantUsecase) Delete(ctx context.Context, userID uuid.UUID, isAdmin bool, id uuid.UUID) error {
	// Get participant
	participant, err := u.participantRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify event exists and check authorization
	event, err := u.eventRepo.FindByID(ctx, participant.EventID)
	if err != nil {
		return err
	}

	// Authorization: event owner or admin only
	if !isAdmin && event.OrganizerID != userID {
		return apperrors.Forbidden("you do not have permission to delete this participant")
	}

	// Delete from repository
	if err := u.participantRepo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}
