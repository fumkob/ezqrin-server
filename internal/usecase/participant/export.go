package participant

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

func (u *participantUsecase) ExportCSV(
	ctx context.Context,
	userID uuid.UUID,
	isAdmin bool,
	eventID uuid.UUID,
) ([]*entity.Participant, error) {
	event, err := u.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	if !isAdmin && event.OrganizerID != userID {
		return nil, apperrors.Forbidden(
			"you do not have permission to export participants for this event",
		)
	}

	participants, err := u.participantRepo.FindAllByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	u.populateDistributionURLs(participants)
	return participants, nil
}
