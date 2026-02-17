package checkin

import (
	"context"
	"fmt"

	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// Cancel cancels (undo) a check-in operation
func (u *checkinUsecase) Cancel(
	ctx context.Context,
	userID uuid.UUID,
	isAdmin bool,
	checkinID uuid.UUID,
) error {
	// Find check-in
	checkin, err := u.checkinRepo.FindByID(ctx, checkinID)
	if err != nil {
		return err
	}

	// Verify event exists and check authorization
	event, err := u.eventRepo.FindByID(ctx, checkin.EventID)
	if err != nil {
		return err
	}

	// Authorization: event owner or admin only
	if !isAdmin && event.OrganizerID != userID {
		return apperrors.Forbidden("you do not have permission to cancel check-ins for this event")
	}

	// Delete check-in
	if err := u.checkinRepo.Delete(ctx, checkinID); err != nil {
		return fmt.Errorf("failed to cancel check-in: %w", err)
	}

	return nil
}
