package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ParticipantRepository implements the ParticipantRepository interface.
type participantRepository struct {
	pool *pgxpool.Pool
}

// NewParticipantRepository creates a new participant repository.
func NewParticipantRepository(pool *pgxpool.Pool) repository.ParticipantRepository {
	return &participantRepository{pool: pool}
}

// Create creates a new participant in the database.
func (r *participantRepository) Create(ctx context.Context, participant *entity.Participant) error {
	if err := participant.Validate(); err != nil {
		return fmt.Errorf("invalid participant: %w", err)
	}

	query := `
		INSERT INTO participants (
			id, event_id, name, email, employee_id, phone, qr_email, status,
			qr_code, qr_code_generated_at, metadata, payment_status, payment_amount,
			payment_date, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	_, err := r.pool.Exec(ctx, query,
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
		participant.Metadata,
		participant.PaymentStatus,
		participant.PaymentAmount,
		participant.PaymentDate,
		participant.CreatedAt,
		participant.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violations
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"unique_event_email\" (SQLSTATE 23505)" {
			return fmt.Errorf("participant with this email already exists for this event: %w", err)
		}
		if err.Error() == "ERROR: duplicate key value violates unique constraint "+
			"\"participants_qr_code_key\" (SQLSTATE 23505)" {
			return fmt.Errorf("QR code already exists: %w", err)
		}
		return fmt.Errorf("failed to insert participant: %w", err)
	}

	return nil
}

// BulkCreate creates multiple participants in the database with optimized performance.
func (r *participantRepository) BulkCreate(ctx context.Context, participants []*entity.Participant) error {
	if len(participants) == 0 {
		return nil
	}

	// Validate all participants first
	for _, p := range participants {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("invalid participant: %w", err)
		}
	}

	// Use pgx batch for optimized bulk insert
	batch := &pgx.Batch{}

	query := `
		INSERT INTO participants (
			id, event_id, name, email, employee_id, phone, qr_email, status,
			qr_code, qr_code_generated_at, metadata, payment_status, payment_amount,
			payment_date, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	for _, p := range participants {
		batch.Queue(query,
			p.ID,
			p.EventID,
			p.Name,
			p.Email,
			p.EmployeeID,
			p.Phone,
			p.QREmail,
			p.Status,
			p.QRCode,
			p.QRCodeGeneratedAt,
			p.Metadata,
			p.PaymentStatus,
			p.PaymentAmount,
			p.PaymentDate,
			p.CreatedAt,
			p.UpdatedAt,
		)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < len(participants); i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert participant batch: %w", err)
		}
	}

	return nil
}

// FindByID retrieves a participant by its unique ID.
func (r *participantRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Participant, error) {
	query := `
		SELECT
			id, event_id, name, email, employee_id, phone, qr_email, status,
			qr_code, qr_code_generated_at, metadata, payment_status, payment_amount,
			payment_date, created_at, updated_at
		FROM participants
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	participant, err := r.scanParticipantFromRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("participant not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find participant: %w", err)
	}

	return participant, nil
}

// FindByEventID retrieves paginated participants for an event with check-in status.
func (r *participantRepository) FindByEventID(
	ctx context.Context,
	eventID uuid.UUID,
	offset, limit int,
) (
	[]*entity.Participant,
	int64,
	error,
) {
	query := `
		SELECT
			p.id, p.event_id, p.name, p.email, p.employee_id, p.phone, p.qr_email, p.status,
			p.qr_code, p.qr_code_generated_at, p.metadata, p.payment_status, p.payment_amount,
			p.payment_date, p.created_at, p.updated_at, c.checked_in_at
		FROM participants p
		LEFT JOIN checkins c ON c.participant_id = p.id AND c.event_id = p.event_id
		WHERE p.event_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	countQuery := `
		SELECT COUNT(*)
		FROM participants
		WHERE event_id = $1
	`

	participants, err := r.queryParticipantsWithCheckin(ctx, query, eventID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.countParticipants(ctx, countQuery, eventID)
	if err != nil {
		return nil, 0, err
	}

	return participants, total, nil
}

// FindByQRCode retrieves a participant by their QR code.
func (r *participantRepository) FindByQRCode(ctx context.Context, qrCode string) (*entity.Participant, error) {
	query := `
		SELECT
			id, event_id, name, email, employee_id, phone, qr_email, status,
			qr_code, qr_code_generated_at, metadata, payment_status, payment_amount,
			payment_date, created_at, updated_at
		FROM participants
		WHERE qr_code = $1
	`

	row := r.pool.QueryRow(ctx, query, qrCode)
	participant, err := r.scanParticipantFromRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("participant not found")
		}
		return nil, fmt.Errorf("failed to find participant by QR code: %w", err)
	}

	return participant, nil
}

// Update updates an existing participant's information.
func (r *participantRepository) Update(ctx context.Context, participant *entity.Participant) error {
	if err := participant.Validate(); err != nil {
		return fmt.Errorf("invalid participant: %w", err)
	}

	query := `
		UPDATE participants
		SET
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

	result, err := r.pool.Exec(ctx, query,
		participant.Name,
		participant.Email,
		participant.EmployeeID,
		participant.Phone,
		participant.QREmail,
		participant.Status,
		participant.Metadata,
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
		return fmt.Errorf("participant not found")
	}

	return nil
}

// Delete deletes a participant from the database.
func (r *participantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM participants
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete participant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("participant not found")
	}

	return nil
}

// Search searches for participants within an event by name, email, or employee_id.
func (r *participantRepository) Search(
	ctx context.Context,
	eventID uuid.UUID,
	query string,
	offset, limit int,
) (
	[]*entity.Participant,
	int64,
	error,
) {
	searchPattern := "%" + query + "%"

	sqlQuery := `
		SELECT
			p.id, p.event_id, p.name, p.email, p.employee_id, p.phone, p.qr_email, p.status,
			p.qr_code, p.qr_code_generated_at, p.metadata, p.payment_status, p.payment_amount,
			p.payment_date, p.created_at, p.updated_at, c.checked_in_at
		FROM participants p
		LEFT JOIN checkins c ON c.participant_id = p.id AND c.event_id = p.event_id
		WHERE p.event_id = $1
		AND (
			p.name ILIKE $2
			OR p.email ILIKE $2
			OR p.employee_id ILIKE $2
		)
		ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4
	`

	countQuery := `
		SELECT COUNT(*)
		FROM participants
		WHERE event_id = $1
		AND (
			name ILIKE $2
			OR email ILIKE $2
			OR employee_id ILIKE $2
		)
	`

	participants, err := r.queryParticipantsWithCheckin(ctx, sqlQuery, eventID, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.countParticipants(ctx, countQuery, eventID, searchPattern)
	if err != nil {
		return nil, 0, err
	}

	return participants, total, nil
}

// ExistsByEmail checks if a participant with the given email exists for an event.
func (r *participantRepository) ExistsByEmail(ctx context.Context, eventID uuid.UUID, email string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM participants
			WHERE event_id = $1 AND email = $2
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, eventID, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check participant existence: %w", err)
	}

	return exists, nil
}

// GetPaymentStats retrieves payment statistics for participants in an event.
func (r *participantRepository) GetPaymentStats(
	ctx context.Context,
	eventID uuid.UUID,
) (
	*repository.ParticipantPaymentStats,
	error,
) {
	query := `
		SELECT
			COUNT(*) as total_participants,
			COUNT(CASE WHEN payment_status = 'paid' THEN 1 END) as paid_participants,
			COUNT(CASE WHEN payment_status = 'unpaid' THEN 1 END) as unpaid_participants,
			COALESCE(SUM(CASE WHEN payment_status = 'paid' THEN payment_amount ELSE 0 END), 0) as total_payment_amount
		FROM participants
		WHERE event_id = $1
	`

	stats := &repository.ParticipantPaymentStats{}
	err := r.pool.QueryRow(ctx, query, eventID).Scan(
		&stats.TotalParticipants,
		&stats.PaidParticipants,
		&stats.UnpaidParticipants,
		&stats.TotalPaymentAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment stats: %w", err)
	}

	return stats, nil
}

// HealthCheck checks the database connection.
func (r *participantRepository) HealthCheck(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// scanParticipantFromRow scans a single row into a Participant entity.
func (r *participantRepository) scanParticipantFromRow(row pgx.Row) (*entity.Participant, error) {
	participant := &entity.Participant{}
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
		&participant.Metadata,
		&participant.PaymentStatus,
		&participant.PaymentAmount,
		&participant.PaymentDate,
		&participant.CreatedAt,
		&participant.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return participant, nil
}

// scanParticipantWithCheckin scans a row into a Participant entity including check-in status.
func (r *participantRepository) scanParticipantWithCheckin(rows pgx.Rows) (*entity.Participant, error) {
	participant := &entity.Participant{}
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
		&participant.Metadata,
		&participant.PaymentStatus,
		&participant.PaymentAmount,
		&participant.PaymentDate,
		&participant.CreatedAt,
		&participant.UpdatedAt,
		&participant.CheckedInAt,
	)
	if err != nil {
		return nil, err
	}
	participant.CheckedIn = participant.CheckedInAt != nil
	return participant, nil
}

// queryParticipantsWithCheckin executes a query and returns participants with check-in status.
func (r *participantRepository) queryParticipantsWithCheckin(
	ctx context.Context,
	query string,
	args ...interface{},
) (
	[]*entity.Participant,
	error,
) {
	const defaultCapacity = 10
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	participants := make([]*entity.Participant, 0, defaultCapacity)
	for rows.Next() {
		participant, err := r.scanParticipantWithCheckin(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, participant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating participants: %w", err)
	}

	return participants, nil
}

// countParticipants counts participants matching the query.
func (r *participantRepository) countParticipants(
	ctx context.Context,
	query string,
	args ...interface{},
) (
	int64,
	error,
) {
	var total int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count participants: %w", err)
	}
	return total, nil
}
