package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ParticipantStatus represents the status of a participant for an event.
type ParticipantStatus string

const (
	// ParticipantStatusTentative means the participant has been invited but hasn't confirmed.
	ParticipantStatusTentative ParticipantStatus = "tentative"
	// ParticipantStatusConfirmed means the participant has confirmed attendance.
	ParticipantStatusConfirmed ParticipantStatus = "confirmed"
	// ParticipantStatusCancelled means the participant cancelled their attendance.
	ParticipantStatusCancelled ParticipantStatus = "cancelled"
	// ParticipantStatusDeclined means the participant declined the invitation.
	ParticipantStatusDeclined ParticipantStatus = "declined"
)

// PaymentStatus represents the payment status of a participant.
// This is separate from ParticipantStatus to allow participants to confirm attendance
// before payment, supporting scenarios like pay-at-door or flexible payment options.
type PaymentStatus string

const (
	// PaymentUnpaid means no payment has been made.
	PaymentUnpaid PaymentStatus = "unpaid"
	// PaymentPaid means payment has been received.
	PaymentPaid PaymentStatus = "paid"
	// PaymentRefunded means payment has been refunded.
	PaymentRefunded PaymentStatus = "refunded"
)

// Validation constants for Participant entity
const (
	ParticipantNameMinLength   = 1
	ParticipantNameMaxLength   = 255
	ParticipantEmailMaxLength  = 255
	ParticipantPhoneMaxLength  = 50
	ParticipantEmployeeIDMax   = 255
	QRCodeLength               = 255
)

// Common validation errors for Participant entity
var (
	ErrParticipantNameRequired    = errors.New("participant name is required")
	ErrParticipantNameTooLong     = errors.New("participant name must not exceed 255 characters")
	ErrParticipantEmailRequired   = errors.New("participant email is required")
	ErrParticipantEmailTooLong    = errors.New("participant email must not exceed 255 characters")
	ErrParticipantEmailInvalid    = errors.New("participant email is invalid")
	ErrParticipantPhoneTooLong    = errors.New("participant phone must not exceed 50 characters")
	ErrParticipantEmployeeIDLong  = errors.New("participant employee_id must not exceed 255 characters")
	ErrParticipantStatusInvalid   = errors.New("invalid participant status")
	ErrParticipantQRCodeRequired  = errors.New("participant qr_code is required")
	ErrParticipantQRCodeTooLong   = errors.New("participant qr_code must not exceed 255 characters")
	ErrParticipantEventIDRequired = errors.New("participant event_id is required")
)

// Participant represents a person registered for an event.
type Participant struct {
	ID               uuid.UUID
	EventID          uuid.UUID
	Name             string
	Email            string
	EmployeeID       *string
	Phone            *string
	QREmail          *string
	Status           ParticipantStatus
	QRCode           string
	QRCodeGeneratedAt time.Time
	Metadata         map[string]any
	PaymentStatus    PaymentStatus
	PaymentAmount    *float64
	PaymentDate      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Validate validates the Participant entity fields.
func (p *Participant) Validate() error {
	if p.EventID == uuid.Nil {
		return ErrParticipantEventIDRequired
	}
	if p.Name == "" {
		return ErrParticipantNameRequired
	}
	if len(p.Name) > ParticipantNameMaxLength {
		return ErrParticipantNameTooLong
	}
	if p.Email == "" {
		return ErrParticipantEmailRequired
	}
	if len(p.Email) > ParticipantEmailMaxLength {
		return ErrParticipantEmailTooLong
	}
	if !isValidEmail(p.Email) {
		return ErrParticipantEmailInvalid
	}
	if p.Phone != nil && len(*p.Phone) > ParticipantPhoneMaxLength {
		return ErrParticipantPhoneTooLong
	}
	if p.EmployeeID != nil && len(*p.EmployeeID) > ParticipantEmployeeIDMax {
		return ErrParticipantEmployeeIDLong
	}
	if p.QRCode == "" {
		return ErrParticipantQRCodeRequired
	}
	if len(p.QRCode) > QRCodeLength {
		return ErrParticipantQRCodeTooLong
	}
	if !p.IsValidStatus() {
		return ErrParticipantStatusInvalid
	}
	return nil
}

// IsValidStatus checks if the participant status is valid.
func (p *Participant) IsValidStatus() bool {
	switch p.Status {
	case ParticipantStatusTentative, ParticipantStatusConfirmed, ParticipantStatusCancelled, ParticipantStatusDeclined:
		return true
	default:
		return false
	}
}

// IsPaid returns true if the participant has paid.
func (p *Participant) IsPaid() bool {
	return p.PaymentStatus == PaymentPaid
}

// IsConfirmed returns true if the participant has confirmed attendance.
func (p *Participant) IsConfirmed() bool {
	return p.Status == ParticipantStatusConfirmed
}

// IsCancelled returns true if the participant cancelled attendance.
func (p *Participant) IsCancelled() bool {
	return p.Status == ParticipantStatusCancelled
}

// IsDeclined returns true if the participant declined the invitation.
func (p *Participant) IsDeclined() bool {
	return p.Status == ParticipantStatusDeclined
}

// isValidEmail performs basic email validation.
func isValidEmail(email string) bool {
	// Basic email validation: must contain @ and at least one dot
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false // Multiple @ symbols
			}
			atIndex = i
		}
	}
	if atIndex == -1 {
		return false // No @ symbol
	}
	if atIndex == 0 || atIndex == len(email)-1 {
		return false // @ at start or end
	}

	// Check domain part contains at least one dot
	domain := email[atIndex+1:]
	hasDot := false
	for _, c := range domain {
		if c == '.' {
			hasDot = true
			break
		}
	}
	return hasDot
}
