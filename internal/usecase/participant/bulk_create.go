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
func (u *participantUsecase) BulkCreate(
	ctx context.Context,
	userID uuid.UUID,
	isAdmin bool,
	input BulkCreateInput,
) (BulkCreateOutput, error) {
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
		if err := u.processSingleParticipant(ctx, i, participantInput, input.EventID, &output); err != nil {
			// Error already recorded in output
			continue
		}
	}

	return output, nil
}

// processSingleParticipant processes a single participant in bulk creation
func (u *participantUsecase) processSingleParticipant(
	ctx context.Context,
	index int,
	input CreateParticipantInput,
	eventID uuid.UUID,
	output *BulkCreateOutput,
) error {
	participant, err := u.buildParticipantEntity(input, eventID)
	if err != nil {
		output.FailedCount++
		output.Errors = append(output.Errors, BulkCreateError{
			Index:   index,
			Email:   input.Email,
			Message: err.Error(),
		})
		return err
	}

	// Save to repository
	if err := u.participantRepo.Create(ctx, participant); err != nil {
		output.FailedCount++
		output.Errors = append(output.Errors, BulkCreateError{
			Index:   index,
			Email:   input.Email,
			Message: fmt.Sprintf("database error: %v", err),
		})
		return err
	}

	output.CreatedCount++
	output.Participants = append(output.Participants, participant)
	return nil
}

// buildParticipantEntity builds a participant entity from input with validation
func (u *participantUsecase) buildParticipantEntity(
	input CreateParticipantInput,
	eventID uuid.UUID,
) (*entity.Participant, error) {
	// Generate participant ID first so it can be embedded in the QR token
	participantID := uuid.New()

	// Generate QR code token with structured format
	qrToken, err := crypto.GenerateParticipantQRToken(eventID, participantID, u.qrHMACSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR token: %w", err)
	}

	// Convert metadata string to json.RawMessage
	var metadata *json.RawMessage
	if input.Metadata != nil {
		raw := json.RawMessage(*input.Metadata)
		metadata = &raw
	}

	// Create participant entity
	now := time.Now()
	participant := &entity.Participant{
		ID:                participantID, // Use pre-generated ID
		EventID:           eventID,
		Name:              input.Name,
		Email:             input.Email,
		QREmail:           input.QREmail,
		EmployeeID:        input.EmployeeID,
		Phone:             input.Phone,
		QRCode:            qrToken,
		QRCodeGeneratedAt: now,
		Status:            input.Status,
		Metadata:          metadata,
		PaymentStatus:     input.PaymentStatus,
		PaymentAmount:     input.PaymentAmount,
		PaymentDate:       input.PaymentDate,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Validate participant
	if err := participant.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return participant, nil
}
