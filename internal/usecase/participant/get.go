package participant

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// GetByID retrieves a participant by ID with authorization check
func (u *participantUsecase) GetByID(ctx context.Context, userID uuid.UUID, isAdmin bool, id uuid.UUID) (*entity.Participant, error) {
	// Get participant
	participant, err := u.participantRepo.FindByID(ctx, id)
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
		return nil, apperrors.Forbidden("you do not have permission to view this participant")
	}

	return participant, nil
}
