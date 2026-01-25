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

// Create creates a new participant with QR code generation
func (u *participantUsecase) Create(ctx context.Context, userID uuid.UUID, isAdmin bool, input CreateParticipantInput) (*entity.Participant, error) {
	// Verify event exists and check authorization
	event, err := u.eventRepo.FindByID(ctx, input.EventID)
	if err != nil {
		return nil, err
	}

	// Authorization: event owner or admin only
	if !isAdmin && event.OrganizerID != userID {
		return nil, apperrors.Forbidden("you do not have permission to add participants to this event")
	}

	// Generate QR code token
	qrToken, err := crypto.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR token: %w", err)
	}

	// Convert metadata string to json.RawMessage if provided
	var metadata *json.RawMessage
	if input.Metadata != nil {
		raw := json.RawMessage(*input.Metadata)
		metadata = &raw
	}

	// Create participant entity
	now := time.Now()
	participant := &entity.Participant{
		ID:                uuid.New(),
		EventID:           input.EventID,
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
		return nil, apperrors.Validation(fmt.Sprintf("participant validation failed: %v", err))
	}

	// Save to repository
	if err := u.participantRepo.Create(ctx, participant); err != nil {
		return nil, err
	}

	return participant, nil
}
