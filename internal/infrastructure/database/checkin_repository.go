package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// percentageMultiplier is used to convert decimal to percentage (0.0-1.0 to 0.0-100.0)
	percentageMultiplier = 100.0
)

// checkinRepository implements the CheckinRepository interface.
type checkinRepository struct {
	pool *pgxpool.Pool
}

// NewCheckinRepository creates a new checkin repository.
func NewCheckinRepository(pool *pgxpool.Pool) repository.CheckinRepository {
	return &checkinRepository{pool: pool}
}

// Create creates a new check-in record with duplicate prevention.
func (r *checkinRepository) Create(ctx context.Context, checkin *entity.Checkin) error {
	if err := checkin.Validate(); err != nil {
		return fmt.Errorf("invalid checkin: %w", err)
	}

	query := `
		INSERT INTO checkins (
			id, event_id, participant_id, checked_in_at, checked_in_by,
			checkin_method, device_info
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`

	_, err := r.pool.Exec(ctx, query,
		checkin.ID,
		checkin.EventID,
		checkin.ParticipantID,
		checkin.CheckedInAt,
		checkin.CheckedInBy,
		checkin.Method,
		checkin.DeviceInfo,
	)
	if err != nil {
		// Check for unique constraint violation (duplicate check-in)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
			strings.Contains(pgErr.ConstraintName, "unique_event_participant_checkin") {
			return entity.ErrCheckinAlreadyExists
		}
		return fmt.Errorf("failed to insert checkin: %w", err)
	}

	return nil
}

// FindByID finds a check-in by its unique ID.
func (r *checkinRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Checkin, error) {
	query := `
		SELECT
			id, event_id, participant_id, checked_in_at, checked_in_by,
			checkin_method, device_info
		FROM checkins
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	checkin, err := r.scanCheckinFromRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find checkin: %w", err)
	}

	return checkin, nil
}

// FindByParticipant finds check-in for a participant.
func (r *checkinRepository) FindByParticipant(ctx context.Context, participantID uuid.UUID) (*entity.Checkin, error) {
	query := `
		SELECT
			id, event_id, participant_id, checked_in_at, checked_in_by,
			checkin_method, device_info
		FROM checkins
		WHERE participant_id = $1
	`

	row := r.pool.QueryRow(ctx, query, participantID)
	checkin, err := r.scanCheckinFromRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find checkin by participant: %w", err)
	}

	return checkin, nil
}

// FindByEvent finds all check-ins for an event with pagination.
func (r *checkinRepository) FindByEvent(
	ctx context.Context,
	eventID uuid.UUID,
	limit, offset int,
) (
	[]*entity.Checkin,
	int64,
	error,
) {
	query := `
		SELECT
			id, event_id, participant_id, checked_in_at, checked_in_by,
			checkin_method, device_info
		FROM checkins
		WHERE event_id = $1
		ORDER BY checked_in_at DESC
		LIMIT $2 OFFSET $3
	`

	countQuery := `
		SELECT COUNT(*)
		FROM checkins
		WHERE event_id = $1
	`

	checkins, err := r.queryCheckins(ctx, query, eventID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.countCheckins(ctx, countQuery, eventID)
	if err != nil {
		return nil, 0, err
	}

	return checkins, total, nil
}

// GetEventStats gets check-in statistics for an event.
func (r *checkinRepository) GetEventStats(ctx context.Context, eventID uuid.UUID) (*repository.CheckinStats, error) {
	query := `
		SELECT
			COUNT(DISTINCT p.id) as total_participants,
			COUNT(DISTINCT c.id) as checked_in_count
		FROM participants p
		LEFT JOIN checkins c ON p.id = c.participant_id AND c.event_id = p.event_id
		WHERE p.event_id = $1
	`

	stats := &repository.CheckinStats{}
	err := r.pool.QueryRow(ctx, query, eventID).Scan(
		&stats.TotalParticipants,
		&stats.CheckedInCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get checkin stats: %w", err)
	}

	// Calculate check-in rate percentage
	if stats.TotalParticipants > 0 {
		stats.CheckinRate = (float64(stats.CheckedInCount) / float64(stats.TotalParticipants)) * percentageMultiplier
	} else {
		stats.CheckinRate = 0.0
	}

	return stats, nil
}

// Delete deletes a check-in (undo check-in operation).
func (r *checkinRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM checkins
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete checkin: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// ExistsByParticipant checks if a participant has already checked in to an event.
func (r *checkinRepository) ExistsByParticipant(ctx context.Context, eventID, participantID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM checkins
			WHERE event_id = $1 AND participant_id = $2
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, eventID, participantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check checkin existence: %w", err)
	}

	return exists, nil
}

// HealthCheck checks the database connection.
func (r *checkinRepository) HealthCheck(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// scanCheckinFromRow scans a single row into a Checkin entity.
func (r *checkinRepository) scanCheckinFromRow(row pgx.Row) (*entity.Checkin, error) {
	checkin := &entity.Checkin{}
	err := row.Scan(
		&checkin.ID,
		&checkin.EventID,
		&checkin.ParticipantID,
		&checkin.CheckedInAt,
		&checkin.CheckedInBy,
		&checkin.Method,
		&checkin.DeviceInfo,
	)
	if err != nil {
		return nil, err
	}
	return checkin, nil
}

// scanCheckin scans a row into a Checkin entity.
func (r *checkinRepository) scanCheckin(rows pgx.Rows) (*entity.Checkin, error) {
	checkin := &entity.Checkin{}
	err := rows.Scan(
		&checkin.ID,
		&checkin.EventID,
		&checkin.ParticipantID,
		&checkin.CheckedInAt,
		&checkin.CheckedInBy,
		&checkin.Method,
		&checkin.DeviceInfo,
	)
	if err != nil {
		return nil, err
	}
	return checkin, nil
}

// queryCheckins executes a query and returns checkins.
func (r *checkinRepository) queryCheckins(
	ctx context.Context,
	query string,
	args ...any,
) (
	[]*entity.Checkin,
	error,
) {
	const defaultCapacity = 10
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query checkins: %w", err)
	}
	defer rows.Close()

	checkins := make([]*entity.Checkin, 0, defaultCapacity)
	for rows.Next() {
		checkin, err := r.scanCheckin(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkin: %w", err)
		}
		checkins = append(checkins, checkin)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating checkins: %w", err)
	}

	return checkins, nil
}

// countCheckins counts checkins matching the query.
func (r *checkinRepository) countCheckins(
	ctx context.Context,
	query string,
	args ...any,
) (
	int64,
	error,
) {
	var total int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count checkins: %w", err)
	}
	return total, nil
}
