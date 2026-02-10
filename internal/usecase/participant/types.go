package participant

import (
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
)

// CreateParticipantInput represents input for creating a participant
type CreateParticipantInput struct {
	EventID       uuid.UUID
	Name          string
	Email         string
	QREmail       *string
	EmployeeID    *string
	Phone         *string
	Status        entity.ParticipantStatus
	Metadata      *string
	PaymentStatus entity.PaymentStatus
	PaymentAmount *float64
	PaymentDate   *time.Time
}

// UpdateParticipantInput represents input for updating a participant
type UpdateParticipantInput struct {
	Name          *string
	Email         *string
	QREmail       *string
	EmployeeID    *string
	Phone         *string
	Status        *entity.ParticipantStatus
	Metadata      *string
	PaymentStatus *entity.PaymentStatus
	PaymentAmount *float64
	PaymentDate   *time.Time
}

// ListParticipantsInput represents input for listing participants
type ListParticipantsInput struct {
	EventID uuid.UUID
	Page    int
	PerPage int
	Sort    string
	Order   string
	Search  string
	Status  *entity.ParticipantStatus
}

// ListParticipantsOutput represents output for listing participants
type ListParticipantsOutput struct {
	Participants []*entity.Participant
	TotalCount   int64
}

// BulkCreateInput represents input for bulk creating participants
type BulkCreateInput struct {
	EventID      uuid.UUID
	Participants []CreateParticipantInput
}

// BulkCreateOutput represents output for bulk creating participants
type BulkCreateOutput struct {
	CreatedCount int
	FailedCount  int
	Participants []*entity.Participant
	Errors       []BulkCreateError
}

// BulkCreateError represents an error during bulk creation
type BulkCreateError struct {
	Index   int
	Email   string
	Message string
}

// QRCodeOutput represents QR code download output
type QRCodeOutput struct {
	Data        []byte
	ContentType string
	Filename    string
}
