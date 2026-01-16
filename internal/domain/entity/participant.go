package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/pkg/validator"
	"github.com/google/uuid"
)

// ParticipantStatus represents the participation status of a participant.
type ParticipantStatus string

const (
	// ParticipantStatusTentative means the participant has registered but not confirmed.
	ParticipantStatusTentative ParticipantStatus = "tentative"
	// ParticipantStatusConfirmed means the participant has confirmed participation.
	ParticipantStatusConfirmed ParticipantStatus = "confirmed"
	// ParticipantStatusCancelled means the participant has cancelled.
	ParticipantStatusCancelled ParticipantStatus = "cancelled"
	// ParticipantStatusDeclined means the participant has declined.
	ParticipantStatusDeclined ParticipantStatus = "declined"
)

// PaymentStatus represents the payment status of a participant.
type PaymentStatus string

const (
	// PaymentUnpaid means no payment has been received.
	PaymentUnpaid PaymentStatus = "unpaid"
	// PaymentPaid means payment has been received.
	PaymentPaid PaymentStatus = "paid"
)

// Validation constants for Participant entity
const (
	ParticipantNameMinLength       = 1
	ParticipantNameMaxLength       = 255
	ParticipantPhoneMaxLength      = 50
	ParticipantEmployeeIDMaxLength = 255
	MaxMetadataSize                = 10240 // 10KB
)

// Common validation errors for Participant entity
var (
	ErrParticipantNameRequired         = errors.New("participant name is required")
	ErrParticipantNameTooLong          = errors.New("participant name must not exceed 255 characters")
	ErrParticipantEmailRequired        = errors.New("participant email is required")
	ErrParticipantEmailInvalid         = errors.New("participant email format is invalid")
	ErrParticipantQRCodeRequired       = errors.New("QR code is required")
	ErrParticipantStatusInvalid        = errors.New("invalid participant status")
	ErrParticipantPhoneTooLong         = errors.New("phone number must not exceed 50 characters")
	ErrParticipantEmployeeIDTooLong    = errors.New("employee ID must not exceed 255 characters")
	ErrParticipantPaymentStatusInvalid = errors.New("invalid payment status")
	ErrParticipantMetadataTooLarge     = errors.New("metadata must not exceed 10KB")
	ErrParticipantEventIDRequired      = errors.New("event ID is required")
)

// Participant represents an event participant.
type Participant struct {
	ID                uuid.UUID
	EventID           uuid.UUID
	Name              string
	Email             string
	EmployeeID        *string // Optional
	Phone             *string // Optional, E.164 format
	QREmail           *string // Optional, alternative email for QR code
	Status            ParticipantStatus
	QRCode            string
	QRCodeGeneratedAt time.Time
	Metadata          *json.RawMessage // Custom participant data (max 10KB)
	PaymentStatus     PaymentStatus
	PaymentAmount     *float64   // Nullable payment amount
	PaymentDate       *time.Time // Nullable payment date
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Validate validates the Participant entity fields.
func (p *Participant) Validate() error {
	if err := p.validateRequiredFields(); err != nil {
		return err
	}
	if err := p.validateOptionalFields(); err != nil {
		return err
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

// IsValidPaymentStatus checks if the payment status is valid.
func (p *Participant) IsValidPaymentStatus() bool {
	switch p.PaymentStatus {
	case PaymentUnpaid, PaymentPaid:
		return true
	default:
		return false
	}
}

// IsTentative returns true if the participant status is tentative.
func (p *Participant) IsTentative() bool {
	return p.Status == ParticipantStatusTentative
}

// IsConfirmed returns true if the participant status is confirmed.
func (p *Participant) IsConfirmed() bool {
	return p.Status == ParticipantStatusConfirmed
}

// IsCancelled returns true if the participant status is cancelled.
func (p *Participant) IsCancelled() bool {
	return p.Status == ParticipantStatusCancelled
}

// IsDeclined returns true if the participant status is declined.
func (p *Participant) IsDeclined() bool {
	return p.Status == ParticipantStatusDeclined
}

// IsPaid returns true if the payment status is paid.
func (p *Participant) IsPaid() bool {
	return p.PaymentStatus == PaymentPaid
}

// IsUnpaid returns true if the payment status is unpaid.
func (p *Participant) IsUnpaid() bool {
	return p.PaymentStatus == PaymentUnpaid
}

// validateRequiredFields validates required fields.
func (p *Participant) validateRequiredFields() error {
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
	if err := validator.ValidateEmail(p.Email); err != nil {
		return ErrParticipantEmailInvalid
	}
	if p.QRCode == "" {
		return ErrParticipantQRCodeRequired
	}
	if !p.IsValidStatus() {
		return ErrParticipantStatusInvalid
	}
	if !p.IsValidPaymentStatus() {
		return ErrParticipantPaymentStatusInvalid
	}
	return nil
}

// validateOptionalFields validates optional fields.
func (p *Participant) validateOptionalFields() error {
	if p.Phone != nil && len(*p.Phone) > ParticipantPhoneMaxLength {
		return ErrParticipantPhoneTooLong
	}
	if p.EmployeeID != nil && len(*p.EmployeeID) > ParticipantEmployeeIDMaxLength {
		return ErrParticipantEmployeeIDTooLong
	}
	if p.Metadata != nil && len(*p.Metadata) > MaxMetadataSize {
		return ErrParticipantMetadataTooLarge
	}
	return nil
}

// String implements the Stringer interface for ParticipantStatus.
func (s ParticipantStatus) String() string {
	return string(s)
}

// Value implements the driver.Valuer interface for database storage.
func (s ParticipantStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// Scan implements the sql.Scanner interface for database reading.
func (s *ParticipantStatus) Scan(value interface{}) error {
	if value == nil {
		*s = ParticipantStatusTentative
		return nil
	}
	if v, ok := value.(string); ok {
		*s = ParticipantStatus(v)
		return nil
	}
	return errors.New("cannot scan ParticipantStatus")
}

// String implements the Stringer interface for PaymentStatus.
func (s PaymentStatus) String() string {
	return string(s)
}

// Value implements the driver.Valuer interface for database storage.
func (s PaymentStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// Scan implements the sql.Scanner interface for database reading.
func (s *PaymentStatus) Scan(value interface{}) error {
	if value == nil {
		*s = PaymentUnpaid
		return nil
	}
	if v, ok := value.(string); ok {
		*s = PaymentStatus(v)
		return nil
	}
	return errors.New("cannot scan PaymentStatus")
}
