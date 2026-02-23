package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// EventRepository implements repository.EventRepository using PostgreSQL
type EventRepository struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewEventRepository creates a new PostgreSQL-backed EventRepository
func NewEventRepository(pool *pgxpool.Pool, log *logger.Logger) repository.EventRepository {
	return &EventRepository{
		pool:   pool,
		logger: log,
	}
}

// Create creates a new event in the database
func (r *EventRepository) Create(ctx context.Context, event *entity.Event) error {
	query := `
		INSERT INTO events (
			id, organizer_id, name, description, start_date, end_date,
			location, timezone, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	q := GetQueryable(ctx, r.pool)
	_, err := q.Exec(ctx, query,
		event.ID,
		event.OrganizerID,
		event.Name,
		event.Description,
		event.StartDate,
		event.EndDate,
		event.Location,
		event.Timezone,
		event.Status,
		event.CreatedAt,
		event.UpdatedAt,
	)
	if err != nil {
		return apperrors.Wrapf(err, "failed to create event")
	}

	r.logger.WithContext(ctx).Info("event created",
		zap.String("event_id", event.ID.String()),
		zap.String("organizer_id", event.OrganizerID.String()),
		zap.String("name", event.Name),
	)

	return nil
}

// FindByID retrieves an event by its unique ID
func (r *EventRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
	query := `
		SELECT
			id, organizer_id, name, description, start_date, end_date,
			location, timezone, status, created_at, updated_at
		FROM events
		WHERE id = $1
	`

	var event entity.Event
	q := GetQueryable(ctx, r.pool)
	err := q.QueryRow(ctx, query, id).Scan(
		&event.ID,
		&event.OrganizerID,
		&event.Name,
		&event.Description,
		&event.StartDate,
		&event.EndDate,
		&event.Location,
		&event.Timezone,
		&event.Status,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("event not found")
		}
		return nil, apperrors.Wrapf(err, "failed to find event by id")
	}

	return &event, nil
}

// List retrieves a paginated and filtered list of events
func (r *EventRepository) List(
	ctx context.Context,
	filter repository.EventListFilter,
	offset, limit int,
) ([]*entity.Event, int64, error) {
	whereSQL, args, argIdx := r.buildListWhereClause(filter)

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events WHERE %s", whereSQL)
	var total int64
	q := GetQueryable(ctx, r.pool)
	err := q.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperrors.Wrapf(err, "failed to count events")
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, organizer_id, name, description, start_date, end_date,
			location, timezone, status, created_at, updated_at
		FROM events
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, apperrors.Wrapf(err, "failed to list events")
	}
	defer rows.Close()

	events := make([]*entity.Event, 0, limit)
	for rows.Next() {
		var event entity.Event
		err := rows.Scan(
			&event.ID,
			&event.OrganizerID,
			&event.Name,
			&event.Description,
			&event.StartDate,
			&event.EndDate,
			&event.Location,
			&event.Timezone,
			&event.Status,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, 0, apperrors.Wrapf(err, "failed to scan event row")
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.Wrapf(err, "error iterating event rows")
	}

	return events, total, nil
}

// Update updates an existing event's information
func (r *EventRepository) Update(ctx context.Context, event *entity.Event) error {
	query := `
		UPDATE events
		SET
			name = $2,
			description = $3,
			start_date = $4,
			end_date = $5,
			location = $6,
			timezone = $7,
			status = $8,
			updated_at = $9
		WHERE id = $1
	`

	q := GetQueryable(ctx, r.pool)
	commandTag, err := q.Exec(ctx, query,
		event.ID,
		event.Name,
		event.Description,
		event.StartDate,
		event.EndDate,
		event.Location,
		event.Timezone,
		event.Status,
		event.UpdatedAt,
	)
	if err != nil {
		return apperrors.Wrapf(err, "failed to update event")
	}

	if commandTag.RowsAffected() == 0 {
		return apperrors.NotFound("event not found")
	}

	r.logger.WithContext(ctx).Info("event updated",
		zap.String("event_id", event.ID.String()),
		zap.String("name", event.Name),
		zap.String("status", string(event.Status)),
	)

	return nil
}

// Delete deletes an event from the database
func (r *EventRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM events WHERE id = $1`

	q := GetQueryable(ctx, r.pool)
	commandTag, err := q.Exec(ctx, query, id)
	if err != nil {
		return apperrors.Wrapf(err, "failed to delete event")
	}

	if commandTag.RowsAffected() == 0 {
		return apperrors.NotFound("event not found")
	}

	r.logger.WithContext(ctx).Info("event deleted",
		zap.String("event_id", id.String()),
	)

	return nil
}

// GetStats retrieves basic statistics for an event
func (r *EventRepository) GetStats(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
	// First check if event exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM events WHERE id = $1)`
	q := GetQueryable(ctx, r.pool)
	if err := q.QueryRow(ctx, checkQuery, id).Scan(&exists); err != nil {
		return nil, apperrors.Wrapf(err, "failed to check event existence")
	}
	if !exists {
		return nil, apperrors.NotFound("event not found")
	}

	// Get active participant count (tentative + confirmed) and checked-in count
	statsQuery := `
		SELECT
			(SELECT COUNT(*) FROM participants
			 WHERE event_id = $1 AND status NOT IN ('cancelled', 'declined')) as total_participants,
			(SELECT COUNT(*) FROM checkins WHERE event_id = $1) as checked_in_count
	`

	stats := &repository.EventStats{}
	err := q.QueryRow(ctx, statsQuery, id).Scan(
		&stats.TotalParticipants,
		&stats.CheckedInCount,
	)
	if err != nil {
		return nil, apperrors.Wrapf(err, "failed to get event statistics")
	}

	// Get participant count by status
	byStatusQuery := `
		SELECT status, COUNT(*) as count
		FROM participants
		WHERE event_id = $1
		GROUP BY status
	`

	rows, err := q.Query(ctx, byStatusQuery, id)
	if err != nil {
		return nil, apperrors.Wrapf(err, "failed to get participant status breakdown")
	}
	defer rows.Close()

	stats.ByStatus = make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, apperrors.Wrapf(err, "failed to scan status row")
		}
		stats.ByStatus[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrapf(err, "failed to iterate status rows")
	}

	return stats, nil
}

// HealthCheck verifies the repository's database connection
func (r *EventRepository) HealthCheck(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *EventRepository) buildListWhereClause(filter repository.EventListFilter) (string, []interface{}, int) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.OrganizerID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("organizer_id = $%d", argIdx))
		args = append(args, *filter.OrganizerID)
		argIdx++
	}

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("start_date >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}

	if filter.EndDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("start_date <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	return strings.Join(whereClauses, " AND "), args, argIdx
}

// Compile-time interface compliance check
var _ repository.EventRepository = (*EventRepository)(nil)
