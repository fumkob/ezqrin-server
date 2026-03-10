package participant

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"sync"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
)

//go:embed templates/qrcode_default.html
var qrCodeEmailTemplate string

const qrEmailSubject = "Your QR Code for %s"

var (
	parsedQRCodeTemplate *template.Template
	parseTemplateOnce    sync.Once
)

type qrCodeEmailData struct {
	ParticipantName string
	EventName       string
	QRCodeURL       string
	ParticipantID   string
}

func renderQRCodeEmail(data qrCodeEmailData) (string, error) {
	var parseErr error
	parseTemplateOnce.Do(func() {
		parsedQRCodeTemplate, parseErr = template.New("qrcode").Parse(qrCodeEmailTemplate)
	})
	if parseErr != nil {
		return "", fmt.Errorf("failed to parse email template: %w", parseErr)
	}

	var buf bytes.Buffer
	if err := parsedQRCodeTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render email template: %w", err)
	}
	return buf.String(), nil
}

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

	// Populate QR distribution URLs for all participants before sending.
	u.populateDistributionURLs(participants)

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

	participants, err := u.participantRepo.FindByIDs(ctx, input.ParticipantIDs)
	if err != nil {
		return nil, err
	}
	// Verify all returned participants belong to the requested event.
	for _, p := range participants {
		if p.EventID != input.EventID {
			return nil, apperrors.NotFound("participant not found in this event")
		}
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
	if p.QRDistributionURL == "" {
		return fmt.Errorf("QRDistributionURL is not configured for participant %s", p.ID)
	}

	body, err := renderQRCodeEmail(qrCodeEmailData{
		ParticipantName: p.Name,
		EventName:       eventName,
		QRCodeURL:       p.QRDistributionURL,
		ParticipantID:   p.ID.String(),
	})
	if err != nil {
		return err
	}

	return u.emailSender.Send(ctx, domainemail.Message{
		To:      dest,
		Subject: fmt.Sprintf(qrEmailSubject, eventName),
		Body:    body,
	})
}
