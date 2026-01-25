package participant

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// Update updates an existing participant with authorization check
func (u *participantUsecase) Update(ctx context.Context, userID uuid.UUID, isAdmin bool, id uuid.UUID, input UpdateParticipantInput) (*entity.Participant, error) {
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
		return nil, apperrors.Forbidden("you do not have permission to update this participant")
	}

	// Apply updates
	if err := applyUpdateInput(participant, input); err != nil {
		return nil, err
	}

	// Validate participant
	if err := participant.Validate(); err != nil {
		return nil, apperrors.Validation(fmt.Sprintf("participant validation failed: %v", err))
	}

	// Update in repository
	if err := u.participantRepo.Update(ctx, participant); err != nil {
		return nil, err
	}

	return participant, nil
}

// applyUpdateInput applies update input fields to participant entity
func applyUpdateInput(participant *entity.Participant, input UpdateParticipantInput) error {
	if input.Name != nil {
		participant.Name = *input.Name
	}
	if input.Email != nil {
		participant.Email = *input.Email
	}
	if input.QREmail != nil {
		participant.QREmail = input.QREmail
	}
	if input.EmployeeID != nil {
		participant.EmployeeID = input.EmployeeID
	}
	if input.Phone != nil {
		participant.Phone = input.Phone
	}
	if input.Status != nil {
		participant.Status = *input.Status
	}
	if input.Metadata != nil {
		// Convert metadata string to json.RawMessage
		metadata := json.RawMessage(*input.Metadata)
		participant.Metadata = &metadata
	}
	if input.PaymentStatus != nil {
		participant.PaymentStatus = *input.PaymentStatus
	}
	if input.PaymentAmount != nil {
		participant.PaymentAmount = input.PaymentAmount
	}
	if input.PaymentDate != nil {
		participant.PaymentDate = input.PaymentDate
	}
	return nil
}
