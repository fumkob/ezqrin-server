package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// CheckinMethod represents the method used for check-in.
type CheckinMethod string

const (
	// CheckinMethodQRCode means the participant checked in via QR code scan.
	CheckinMethodQRCode CheckinMethod = "qrcode"
	// CheckinMethodManual means the participant was manually checked in by staff.
	CheckinMethodManual CheckinMethod = "manual"
)

// Common validation errors for Checkin entity
var (
	ErrCheckinEventIDRequired       = errors.New("event ID is required")
	ErrCheckinParticipantIDRequired = errors.New("participant ID is required")
	ErrCheckinMethodInvalid         = errors.New("invalid checkin method")
	ErrCheckinAlreadyExists         = errors.New("participant has already checked in")
)

// Checkin represents a participant check-in record.
type Checkin struct {
	ID            uuid.UUID
	EventID       uuid.UUID
	ParticipantID uuid.UUID
	CheckedInAt   time.Time
	CheckedInBy   *uuid.UUID // Nullable - can be NULL for self-service kiosks
	Method        CheckinMethod
	DeviceInfo    *json.RawMessage // JSONB for device metadata (OS, browser, app version, etc.)
}

// Validate validates the Checkin entity fields.
func (c *Checkin) Validate() error {
	if c.EventID == uuid.Nil {
		return ErrCheckinEventIDRequired
	}
	if c.ParticipantID == uuid.Nil {
		return ErrCheckinParticipantIDRequired
	}
	if !c.IsValidMethod() {
		return ErrCheckinMethodInvalid
	}
	return nil
}

// IsValidMethod checks if the checkin method is valid.
func (c *Checkin) IsValidMethod() bool {
	switch c.Method {
	case CheckinMethodQRCode, CheckinMethodManual:
		return true
	default:
		return false
	}
}

// IsQRCodeMethod returns true if the check-in was performed via QR code scan.
func (c *Checkin) IsQRCodeMethod() bool {
	return c.Method == CheckinMethodQRCode
}

// IsManualMethod returns true if the check-in was performed manually by staff.
func (c *Checkin) IsManualMethod() bool {
	return c.Method == CheckinMethodManual
}

// String implements the Stringer interface for CheckinMethod.
func (m CheckinMethod) String() string {
	return string(m)
}

// Value implements the driver.Valuer interface for database storage.
func (m CheckinMethod) Value() (driver.Value, error) {
	return string(m), nil
}

// Scan implements the sql.Scanner interface for database reading.
func (m *CheckinMethod) Scan(value any) error {
	if value == nil {
		*m = CheckinMethodQRCode // Default to QR code
		return nil
	}
	if v, ok := value.(string); ok {
		*m = CheckinMethod(v)
		return nil
	}
	return errors.New("cannot scan CheckinMethod")
}
