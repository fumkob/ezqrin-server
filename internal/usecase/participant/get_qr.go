package participant

import (
	"context"
	"fmt"

	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// GetQRCode generates and returns a QR code for a participant
func (u *participantUsecase) GetQRCode(ctx context.Context, userID uuid.UUID, isAdmin bool, id uuid.UUID, format string, size int) (QRCodeOutput, error) {
	// Get participant
	participant, err := u.participantRepo.FindByID(ctx, id)
	if err != nil {
		return QRCodeOutput{}, err
	}

	// Verify event exists and check authorization
	event, err := u.eventRepo.FindByID(ctx, participant.EventID)
	if err != nil {
		return QRCodeOutput{}, err
	}

	// Authorization: event owner or admin only
	if !isAdmin && event.OrganizerID != userID {
		return QRCodeOutput{}, apperrors.Forbidden("you do not have permission to download QR code for this participant")
	}

	// Validate format parameter
	if format != "png" && format != "svg" {
		return QRCodeOutput{}, apperrors.BadRequest("invalid format: must be 'png' or 'svg'")
	}

	// Validate size parameter for PNG
	if format == "png" {
		if size < 100 || size > 2000 {
			return QRCodeOutput{}, apperrors.BadRequest("invalid size: must be between 100 and 2000 pixels")
		}
	}

	var data []byte
	var contentType string

	// Generate QR code based on format
	if format == "png" {
		data, err = u.qrGenerator.GeneratePNG(ctx, participant.QRCode, size)
		if err != nil {
			return QRCodeOutput{}, fmt.Errorf("failed to generate PNG QR code: %w", err)
		}
		contentType = "image/png"
	} else {
		// SVG format
		svgString, err := u.qrGenerator.GenerateSVG(ctx, participant.QRCode, size)
		if err != nil {
			return QRCodeOutput{}, fmt.Errorf("failed to generate SVG QR code: %w", err)
		}
		data = []byte(svgString)
		contentType = "image/svg+xml"
	}

	// Generate filename
	filename := fmt.Sprintf("participant-%s-qr.%s", participant.Name, format)

	return QRCodeOutput{
		Data:        data,
		ContentType: contentType,
		Filename:    filename,
	}, nil
}
