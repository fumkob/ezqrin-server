package participant

import (
	"context"
	"fmt"

	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

// GetQRCode generates and returns a QR code for a participant
func (u *participantUsecase) GetQRCode(
	ctx context.Context,
	userID uuid.UUID,
	isAdmin bool,
	id uuid.UUID,
	format string,
	size int,
) (QRCodeOutput, error) {
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
		return QRCodeOutput{}, apperrors.Forbidden(
			"you do not have permission to download QR code for this participant",
		)
	}

	// Validate QR code parameters
	if err := validateQRCodeParams(format, size); err != nil {
		return QRCodeOutput{}, err
	}

	// Generate QR code
	data, contentType, err := u.generateQRCodeData(ctx, participant.QRCode, format, size)
	if err != nil {
		return QRCodeOutput{}, err
	}

	// Generate filename
	filename := fmt.Sprintf("participant-%s-qr.%s", participant.Name, format)

	return QRCodeOutput{
		Data:        data,
		ContentType: contentType,
		Filename:    filename,
	}, nil
}

// validateQRCodeParams validates format and size parameters for QR code generation
func validateQRCodeParams(format string, size int) error {
	if format != "png" && format != "svg" {
		return apperrors.BadRequest("invalid format: must be 'png' or 'svg'")
	}

	if format == "png" && (size < 100 || size > 2000) {
		return apperrors.BadRequest("invalid size: must be between 100 and 2000 pixels")
	}

	return nil
}

// generateQRCodeData generates QR code data based on format
func (u *participantUsecase) generateQRCodeData(
	ctx context.Context,
	qrCode, format string,
	size int,
) ([]byte, string, error) {
	if format == "png" {
		data, err := u.qrGenerator.GeneratePNG(ctx, qrCode, size)
		if err != nil {
			return nil, "", fmt.Errorf("failed to generate PNG QR code: %w", err)
		}
		return data, "image/png", nil
	}

	// SVG format
	svgString, err := u.qrGenerator.GenerateSVG(ctx, qrCode, size)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate SVG QR code: %w", err)
	}
	return []byte(svgString), "image/svg+xml", nil
}
