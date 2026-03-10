package participant

import (
	"context"
	"fmt"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

const qrEmailImageSize = 256

// SendQRCodes sends QR codes to event participants via email.
func (u *participantUsecase) SendQRCodes(
	ctx context.Context,
	userID uuid.UUID,
	isAdmin bool,
	input SendQRCodesInput,
) (SendQRCodesOutput, error) {
	// Validate that at least one target is specified.
	if !input.SendToAll && len(input.ParticipantIDs) == 0 {
		return SendQRCodesOutput{}, apperrors.BadRequest(
			"either participant_ids or send_to_all=true must be provided",
		)
	}

	// Fetch the event and verify the caller has permission.
	event, err := u.eventRepo.FindByID(ctx, input.EventID)
	if err != nil {
		return SendQRCodesOutput{}, err
	}
	if !isAdmin && event.OrganizerID != userID {
		return SendQRCodesOutput{}, apperrors.Forbidden(
			"you do not have permission to send QR codes for this event",
		)
	}

	// Resolve the target participants.
	participants, err := u.resolveParticipants(ctx, input)
	if err != nil {
		return SendQRCodesOutput{}, err
	}

	// Send QR code emails and collect failures.
	var failures []SendQRCodeFailure
	sentCount := 0

	for _, p := range participants {
		dest := destinationEmail(p)
		if err := u.sendQRCodeEmail(ctx, p, dest, event.Name); err != nil {
			failures = append(failures, SendQRCodeFailure{
				ParticipantID: p.ID,
				Email:         dest,
				Reason:        err.Error(),
			})
			continue
		}
		sentCount++
	}

	return SendQRCodesOutput{
		SentCount:   sentCount,
		FailedCount: len(failures),
		Total:       len(participants),
		Failures:    failures,
	}, nil
}

// resolveParticipants returns the participants to send to based on the input.
func (u *participantUsecase) resolveParticipants(
	ctx context.Context,
	input SendQRCodesInput,
) ([]*entity.Participant, error) {
	if input.SendToAll {
		return u.participantRepo.FindAllByEventID(ctx, input.EventID)
	}

	participants := make([]*entity.Participant, 0, len(input.ParticipantIDs))
	for _, id := range input.ParticipantIDs {
		p, err := u.participantRepo.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if p.EventID != input.EventID {
			return nil, apperrors.NotFound("participant not found in this event")
		}
		participants = append(participants, p)
	}
	return participants, nil
}

// destinationEmail returns the email address to send to for a participant.
func destinationEmail(p *entity.Participant) string {
	if p.QREmail != nil && *p.QREmail != "" {
		return *p.QREmail
	}
	return p.Email
}

// sendQRCodeEmail sends a single QR code email to a participant.
// Returns an error if sending failed, or nil on success.
func (u *participantUsecase) sendQRCodeEmail(ctx context.Context, p *entity.Participant, dest, eventName string) error {
	qrData, err := u.qrGenerator.GeneratePNG(ctx, p.QRCode, qrEmailImageSize)
	if err != nil {
		return fmt.Errorf("QR code generation failed: %w", err)
	}

	return u.emailSender.Send(ctx, domainemail.Message{
		To:      dest,
		Subject: fmt.Sprintf("Your QR Code for %s", eventName),
		Body:    buildDefaultEmailBody(p.Name, eventName, p.ID.String()),
		Attachments: []domainemail.Attachment{
			{
				Filename:    "qrcode.png",
				ContentType: "image/png",
				Data:        qrData,
				ContentID:   "qrcode",
			},
		},
	})
}

// buildDefaultEmailBody generates a simple HTML email with an embedded QR code image.
func buildDefaultEmailBody(participantName, eventName, participantID string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"/></head>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:20px;">
  <h2>Your QR Code for %s</h2>
  <p>Hello %s,</p>
  <p>Please show the QR code below at the check-in desk on the day of the event.</p>
  <div style="text-align:center;margin:30px 0;">
    <img src="cid:qrcode" alt="QR Code" width="256" height="256"/>
  </div>
  <p style="color:#666;font-size:12px;">Participant ID: %s</p>
  <hr/>
  <p style="color:#999;font-size:11px;">This email was sent by ezQRin. Please do not reply.</p>
</body>
</html>`, eventName, participantName, participantID)
}
