package checkin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// CheckIn executes the check-in operation for a participant
func (u *checkinUsecase) CheckIn(
	ctx context.Context,
	userID uuid.UUID,
	isAdmin bool,
	input CheckInInput,
) (*CheckInOutput, error) {
	// Verify event exists and check authorization
	event, err := u.eventRepo.FindByID(ctx, input.EventID)
	if err != nil {
		return nil, err
	}

	// Authorization check for manual check-in
	if err := u.checkManualCheckInAuth(input.Method, isAdmin, event.OrganizerID, userID); err != nil {
		return nil, err
	}

	// Find participant based on check-in method
	participant, err := u.findParticipantForCheckIn(ctx, input)
	if err != nil {
		return nil, err
	}

	// Verify participant belongs to this event
	if participant.EventID != input.EventID {
		return nil, apperrors.BadRequest("participant does not belong to this event")
	}

	// Verify participant is active (not cancelled or declined)
	if participant.IsCancelled() || participant.IsDeclined() {
		return nil, apperrors.BadRequest(
			fmt.Sprintf("cannot check in: participant status is %s", participant.Status),
		)
	}

	// Check for duplicate check-in
	if err := u.checkDuplicateCheckIn(ctx, input.EventID, participant.ID); err != nil {
		return nil, err
	}

	// Create and save check-in record
	checkin, err := u.createCheckinRecord(input, participant.ID)
	if err != nil {
		return nil, err
	}

	if err := u.checkinRepo.Create(ctx, checkin); err != nil {
		return nil, u.handleCheckinCreateError(err)
	}

	return u.buildCheckInOutput(checkin, participant), nil
}

// checkManualCheckInAuth checks authorization for manual check-in
func (u *checkinUsecase) checkManualCheckInAuth(
	method entity.CheckinMethod,
	isAdmin bool,
	organizerID, userID uuid.UUID,
) error {
	if method == entity.CheckinMethodManual && !isAdmin && organizerID != userID {
		return apperrors.Forbidden(
			"you do not have permission to manually check in participants for this event",
		)
	}
	return nil
}

// findParticipantForCheckIn finds the participant based on check-in method
func (u *checkinUsecase) findParticipantForCheckIn(
	ctx context.Context,
	input CheckInInput,
) (*entity.Participant, error) {
	if input.Method == entity.CheckinMethodQRCode {
		return u.findParticipantByQRCode(ctx, input)
	}
	if input.Method == entity.CheckinMethodManual {
		return u.findParticipantByID(ctx, input)
	}
	return nil, apperrors.BadRequest("invalid check-in method")
}

// findParticipantByQRCode finds participant by QR code
func (u *checkinUsecase) findParticipantByQRCode(
	ctx context.Context,
	input CheckInInput,
) (*entity.Participant, error) {
	if input.QRCode == nil || *input.QRCode == "" {
		return nil, apperrors.BadRequest("QR code is required for QR code check-in")
	}
	participant, err := u.participantRepo.FindByQRCode(ctx, *input.QRCode)
	if err != nil {
		return nil, apperrors.NotFound("invalid QR code or participant not found")
	}
	return participant, nil
}

// findParticipantByID finds participant by ID
func (u *checkinUsecase) findParticipantByID(
	ctx context.Context,
	input CheckInInput,
) (*entity.Participant, error) {
	if input.ParticipantID == nil {
		return nil, apperrors.BadRequest("participant ID is required for manual check-in")
	}
	return u.participantRepo.FindByID(ctx, *input.ParticipantID)
}

// checkDuplicateCheckIn checks if participant has already checked in
func (u *checkinUsecase) checkDuplicateCheckIn(
	ctx context.Context,
	eventID, participantID uuid.UUID,
) error {
	exists, err := u.checkinRepo.ExistsByParticipant(ctx, eventID, participantID)
	if err != nil {
		return fmt.Errorf("failed to check existing check-in: %w", err)
	}
	if exists {
		return apperrors.Conflict("participant has already checked in")
	}
	return nil
}

// createCheckinRecord creates a check-in entity
func (u *checkinUsecase) createCheckinRecord(
	input CheckInInput,
	participantID uuid.UUID,
) (*entity.Checkin, error) {
	deviceInfo, err := convertDeviceInfo(input.DeviceInfo)
	if err != nil {
		return nil, apperrors.Validation(fmt.Sprintf("invalid device info: %v", err))
	}

	// uuid.Nil の場合は CheckedInBy を nil に設定（セルフサービスキオスク対応）
	var checkedInBy *uuid.UUID
	if input.CheckedInBy != uuid.Nil {
		id := input.CheckedInBy
		checkedInBy = &id
	}

	checkin := &entity.Checkin{
		ID:            uuid.New(),
		EventID:       input.EventID,
		ParticipantID: participantID,
		CheckedInAt:   time.Now(),
		CheckedInBy:   checkedInBy,
		Method:        input.Method,
		DeviceInfo:    deviceInfo,
	}

	if err := checkin.Validate(); err != nil {
		return nil, apperrors.Validation(fmt.Sprintf("check-in validation failed: %v", err))
	}

	return checkin, nil
}

// convertDeviceInfo converts device info map to json.RawMessage
func convertDeviceInfo(deviceInfo map[string]any) (*json.RawMessage, error) {
	if deviceInfo == nil {
		return nil, nil
	}
	data, err := json.Marshal(deviceInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device info: %w", err)
	}
	raw := json.RawMessage(data)
	return &raw, nil
}

// handleCheckinCreateError handles check-in creation errors
func (u *checkinUsecase) handleCheckinCreateError(err error) error {
	if errors.Is(err, entity.ErrCheckinAlreadyExists) {
		return apperrors.Conflict("participant has already checked in")
	}
	return fmt.Errorf("failed to create check-in: %w", err)
}

// buildCheckInOutput builds the check-in output
func (u *checkinUsecase) buildCheckInOutput(
	checkin *entity.Checkin,
	participant *entity.Participant,
) *CheckInOutput {
	return &CheckInOutput{
		ID:                    checkin.ID,
		EventID:               checkin.EventID,
		ParticipantID:         participant.ID,
		ParticipantName:       participant.Name,
		ParticipantEmail:      participant.Email,
		ParticipantEmployeeID: participant.EmployeeID,
		CheckedInAt:           checkin.CheckedInAt,
		CheckedInBy:           checkin.CheckedInBy,
		Method:                checkin.Method,
	}
}
