package participant

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// BulkCreate creates multiple participants with partial success support
func (u *participantUsecase) BulkCreate(ctx context.Context, userID uuid.UUID, isAdmin bool, input BulkCreateInput) (BulkCreateOutput, error) {
	// Verify event exists and check authorization
	event, err := u.eventRepo.FindByID(ctx, input.EventID)
	if err != nil {
		return BulkCreateOutput{}, err
	}

	// Authorization: event owner or admin only
	if !isAdmin && event.OrganizerID != userID {
		return BulkCreateOutput{}, apperrors.Forbidden("you do not have permission to add participants to this event")
	}

	// Initialize output
	output := BulkCreateOutput{
		Participants: make([]*entity.Participant, 0, len(input.Participants)),
		Errors:       make([]BulkCreateError, 0),
	}

	// Process each participant
	for i, participantInput := range input.Participants {
		// Generate QR code token
		qrToken, err := crypto.GenerateToken()
		if err != nil {
			output.FailedCount++
			output.Errors = append(output.Errors, BulkCreateError{
				Index:   i,
				Email:   participantInput.Email,
				Message: fmt.Sprintf("failed to generate QR token: %v", err),
			})
			continue
		}

		// Convert metadata string to json.RawMessage
		var metadata *json.RawMessage
		if participantInput.Metadata != nil {
			raw := json.RawMessage(*participantInput.Metadata)
			metadata = &raw
		}

		// Create participant entity
		now := time.Now()
		participant := &entity.Participant{
			ID:                uuid.New(),
			EventID:           input.EventID,
			Name:              participantInput.Name,
			Email:             participantInput.Email,
			QREmail:           participantInput.QREmail,
			EmployeeID:        participantInput.EmployeeID,
			Phone:             participantInput.Phone,
			QRCode:            qrToken,
			QRCodeGeneratedAt: now,
			Status:            participantInput.Status,
			Metadata:          metadata,
			PaymentStatus:     participantInput.PaymentStatus,
			PaymentAmount:     participantInput.PaymentAmount,
			PaymentDate:       participantInput.PaymentDate,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		// Validate participant
		if err := participant.Validate(); err != nil {
			output.FailedCount++
			output.Errors = append(output.Errors, BulkCreateError{
				Index:   i,
				Email:   participantInput.Email,
				Message: fmt.Sprintf("validation failed: %v", err),
			})
			continue
		}

		// Save to repository
		if err := u.participantRepo.Create(ctx, participant); err != nil {
			output.FailedCount++
			output.Errors = append(output.Errors, BulkCreateError{
				Index:   i,
				Email:   participantInput.Email,
				Message: fmt.Sprintf("database error: %v", err),
			})
			continue
		}

		output.CreatedCount++
		output.Participants = append(output.Participants, participant)
	}

	return output, nil
}
