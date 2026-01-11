package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Participant repository errors.
var (
	ErrParticipantNotFound  = errors.New("participant not found")
	ErrDuplicateEmail       = errors.New("email already exists for this event")
	ErrDuplicateQRCode      = errors.New("qr code already exists")
)

// ParticipantRepositoryImpl implements the ParticipantRepository interface.
type ParticipantRepositoryImpl struct {
	db *pgxpool.Pool
}

// NewParticipantRepository creates a new instance of ParticipantRepositoryImpl.
func NewParticipantRepository(db *pgxpool.Pool) repository.ParticipantRepository {
	return &ParticipantRepositoryImpl{db: db}
}

// HealthCheck verifies the database connection is accessible.
func (r *ParticipantRepositoryImpl) HealthCheck(ctx context.Context) error {
	return r.db.Ping(ctx)
}

// Create creates a new participant in the database.
func (r *ParticipantRepositoryImpl) Create(ctx context.Context, participant *entity.Participant) error {
	if err := participant.Validate(); err != nil {
		return fmt.Errorf("invalid participant: %w", err)
	}

	metadataJSON, err := json.Marshal(participant.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO participants (
			id, event_id, name, email, employee_id, phone, qr_email,
			status, qr_code, qr_code_generated_at, metadata,
			payment_status, payment_amount, payment_date,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14,
			$15, $16
		)
	`

	_, err = r.db.Exec(ctx, query,
		participant.ID,
		participant.EventID,
		participant.Name,
		participant.Email,
		participant.EmployeeID,
		participant.Phone,
		participant.QREmail,
		participant.Status,
		participant.QRCode,
		participant.QRCodeGeneratedAt,
		metadataJSON,
		participant.PaymentStatus,
		participant.PaymentAmount,
		participant.PaymentDate,
		participant.CreatedAt,
		participant.UpdatedAt,
	)

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "unique_event_email") {
			return ErrDuplicateEmail
		}
		if strings.Contains(errMsg, "participants_qr_code_key") {
			return ErrDuplicateQRCode
		}
		return fmt.Errorf("failed to create participant: %w", err)
	}

	return nil
}

// CreateBulk creates multiple participants in a single operation.
func (r *ParticipantRepositoryImpl) CreateBulk(ctx context.Context, participants []*entity.Participant) (int64, error) {
	if len(participants) == 0 {
		return 0, nil
	}

	// Validate all participants
	for _, p := range participants {
		if err := p.Validate(); err != nil {
			return 0, fmt.Errorf("invalid participant: %w", err)
		}
	}

	batch := &pgx.Batch{}

	for _, participant := range participants {
		metadataJSON, err := json.Marshal(participant.Metadata)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		query := `
			INSERT INTO participants (
				id, event_id, name, email, employee_id, phone, qr_email,
				status, qr_code, qr_code_generated_at, metadata,
				payment_status, payment_amount, payment_date,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11,
				$12, $13, $14,
				$15, $16
			)
		`

		batch.Queue(query,
			participant.ID,
			participant.EventID,
			participant.Name,
			participant.Email,
			participant.EmployeeID,
			participant.Phone,
			participant.QREmail,
			participant.Status,
			participant.QRCode,
			participant.QRCodeGeneratedAt,
			metadataJSON,
			participant.PaymentStatus,
			participant.PaymentAmount,
			participant.PaymentDate,
			participant.CreatedAt,
			participant.UpdatedAt,
		)
	}

	results := r.db.SendBatch(ctx, batch)
	defer results.Close()

	var count int64
	for i := 0; i < len(participants); i++ {
		_, err := results.Exec()
		if err != nil {
			return count, fmt.Errorf("failed to create participant at index %d: %w", i, err)
		}
		count++
	}

	return count, nil
}

// FindByID retrieves a participant by ID.
func (r *ParticipantRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.Participant, error) {
	return r.findOne(ctx, "WHERE id = $1", id)
}

// FindByQRCode retrieves a participant by QR code.
func (r *ParticipantRepositoryImpl) FindByQRCode(ctx context.Context, qrCode string) (*entity.Participant, error) {
	return r.findOne(ctx, "WHERE qr_code = $1", qrCode)
}

// FindByEvent retrieves all participants for an event.
func (r *ParticipantRepositoryImpl) FindByEvent(ctx context.Context, eventID uuid.UUID) ([]*entity.Participant, error) {
	query := `
		SELECT id, event_id, name, email, employee_id, phone, qr_email,
		       status, qr_code, qr_code_generated_at, metadata,
		       payment_status, payment_amount, payment_date,
		       created_at, updated_at
		FROM participants
		WHERE event_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	var participants []*entity.Participant
	for rows.Next() {
		participant, err := r.scanParticipant(rows)
		if err != nil {
			return nil, err
		}
		participants = append(participants, participant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return participants, nil
}

// List retrieves a paginated and filtered list of participants.
func (r *ParticipantRepositoryImpl) List(ctx context.Context, filter repository.ParticipantListFilter, offset, limit int) ([]*entity.Participant, int64, error) {
	whereConditions := "WHERE event_id = $1"
	args := []any{filter.EventID}
	argIndex := 2

	if filter.Status != nil {
		whereConditions += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.PaymentStatus != nil {
		whereConditions += fmt.Sprintf(" AND payment_status = $%d", argIndex)
		args = append(args, *filter.PaymentStatus)
		argIndex++
	}

	if filter.Search != "" {
		whereConditions += fmt.Sprintf(` AND (name ILIKE $%d OR email ILIKE $%d OR employee_id ILIKE $%d)`, argIndex, argIndex+1, argIndex+2)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM participants " + whereConditions
	var totalCount int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count participants: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, event_id, name, email, employee_id, phone, qr_email,
		       status, qr_code, qr_code_generated_at, metadata,
		       payment_status, payment_amount, payment_date,
		       created_at, updated_at
		FROM participants
		` + whereConditions + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	var participants []*entity.Participant
	for rows.Next() {
		participant, err := r.scanParticipant(rows)
		if err != nil {
			return nil, 0, err
		}
		participants = append(participants, participant)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	return participants, totalCount, nil
}

// Update updates a participant.
func (r *ParticipantRepositoryImpl) Update(ctx context.Context, participant *entity.Participant) error {
	if err := participant.Validate(); err != nil {
		return fmt.Errorf("invalid participant: %w", err)
	}

	metadataJSON, err := json.Marshal(participant.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE participants SET
			name = $1,
			email = $2,
			employee_id = $3,
			phone = $4,
			qr_email = $5,
			status = $6,
			metadata = $7,
			payment_status = $8,
			payment_amount = $9,
			payment_date = $10,
			updated_at = $11
		WHERE id = $12
	`

	result, err := r.db.Exec(ctx, query,
		participant.Name,
		participant.Email,
		participant.EmployeeID,
		participant.Phone,
		participant.QREmail,
		participant.Status,
		metadataJSON,
		participant.PaymentStatus,
		participant.PaymentAmount,
		participant.PaymentDate,
		participant.UpdatedAt,
		participant.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}

	return nil
}

// Delete deletes a participant.
func (r *ParticipantRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM participants WHERE id = $1"

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete participant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}

	return nil
}

// ExistsByEmail checks if a participant with the given email exists for an event.
func (r *ParticipantRepositoryImpl) ExistsByEmail(ctx context.Context, eventID uuid.UUID, email string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM participants WHERE event_id = $1 AND email = $2)"

	var exists bool
	err := r.db.QueryRow(ctx, query, eventID, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// GetParticipantStats retrieves statistics for participants of an event.
func (r *ParticipantRepositoryImpl) GetParticipantStats(ctx context.Context, eventID uuid.UUID) (*repository.ParticipantStats, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			COUNT(CASE WHEN status = 'confirmed' THEN 1 END) as confirmed_count,
			COUNT(CASE WHEN status = 'tentative' THEN 1 END) as tentative_count,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled_count,
			COUNT(CASE WHEN status = 'declined' THEN 1 END) as declined_count,
			COUNT(CASE WHEN payment_status = 'paid' THEN 1 END) as paid_count,
			COUNT(CASE WHEN payment_status = 'unpaid' THEN 1 END) as unpaid_count,
			COALESCE(SUM(CASE WHEN payment_status = 'paid' THEN payment_amount ELSE 0 END), 0) as total_payment_amount
		FROM participants
		WHERE event_id = $1
	`

	stats := &repository.ParticipantStats{}
	err := r.db.QueryRow(ctx, query, eventID).Scan(
		&stats.TotalCount,
		&stats.ConfirmedCount,
		&stats.TentativeCount,
		&stats.CancelledCount,
		&stats.DeclinedCount,
		&stats.PaidCount,
		&stats.UnpaidCount,
		&stats.TotalPaymentAmount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get participant stats: %w", err)
	}

	return stats, nil
}

// findOne is a helper method to find a single participant.
func (r *ParticipantRepositoryImpl) findOne(ctx context.Context, whereClause string, args ...any) (*entity.Participant, error) {
	query := `
		SELECT id, event_id, name, email, employee_id, phone, qr_email,
		       status, qr_code, qr_code_generated_at, metadata,
		       payment_status, payment_amount, payment_date,
		       created_at, updated_at
		FROM participants
		` + whereClause

	row := r.db.QueryRow(ctx, query, args...)

	participant := &entity.Participant{}
	var metadataJSON []byte

	err := row.Scan(
		&participant.ID,
		&participant.EventID,
		&participant.Name,
		&participant.Email,
		&participant.EmployeeID,
		&participant.Phone,
		&participant.QREmail,
		&participant.Status,
		&participant.QRCode,
		&participant.QRCodeGeneratedAt,
		&metadataJSON,
		&participant.PaymentStatus,
		&participant.PaymentAmount,
		&participant.PaymentDate,
		&participant.CreatedAt,
		&participant.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrParticipantNotFound
		}
		return nil, fmt.Errorf("failed to scan participant: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &participant.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return participant, nil
}

// scanParticipant scans a row into a Participant entity.
func (r *ParticipantRepositoryImpl) scanParticipant(rows pgx.Rows) (*entity.Participant, error) {
	participant := &entity.Participant{}
	var metadataJSON []byte

	err := rows.Scan(
		&participant.ID,
		&participant.EventID,
		&participant.Name,
		&participant.Email,
		&participant.EmployeeID,
		&participant.Phone,
		&participant.QREmail,
		&participant.Status,
		&participant.QRCode,
		&participant.QRCodeGeneratedAt,
		&metadataJSON,
		&participant.PaymentStatus,
		&participant.PaymentAmount,
		&participant.PaymentDate,
		&participant.CreatedAt,
		&participant.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan participant: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &participant.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return participant, nil
}
