package checkin

import (
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
)

// CheckInInput represents input for checking in a participant
type CheckInInput struct {
	EventID       uuid.UUID
	Method        entity.CheckinMethod
	QRCode        *string
	ParticipantID *uuid.UUID
	CheckedInBy   uuid.UUID
	DeviceInfo    map[string]interface{}
}

// CheckInOutput represents output after checking in
type CheckInOutput struct {
	ID               uuid.UUID
	EventID          uuid.UUID
	ParticipantID    uuid.UUID
	ParticipantName  string
	ParticipantEmail string
	CheckedInAt      time.Time
	CheckedInBy      *uuid.UUID
	Method           entity.CheckinMethod
}

// CheckInStatusOutput represents check-in status for a participant
type CheckInStatusOutput struct {
	ParticipantID uuid.UUID
	IsCheckedIn   bool
	CheckIn       *CheckInOutput
}

// ListCheckInsInput represents input for listing check-ins
type ListCheckInsInput struct {
	EventID uuid.UUID
	Page    int
	PerPage int
	Sort    string
	Order   string
}

// ListCheckInsOutput represents output for listing check-ins
type ListCheckInsOutput struct {
	CheckIns   []*CheckInOutput
	TotalCount int64
}
