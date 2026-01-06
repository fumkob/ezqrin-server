package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PostgreSQL error codes
const (
	pgErrCodeUniqueViolation     = "23505" // unique_violation
	pgErrCodeForeignKeyViolation = "23503" // foreign_key_violation
)

// UserRepository implements repository.UserRepository using PostgreSQL
type UserRepository struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewUserRepository creates a new PostgreSQL-backed UserRepository
func NewUserRepository(pool *pgxpool.Pool, log *logger.Logger) repository.UserRepository {
	return &UserRepository{
		pool:   pool,
		logger: log,
	}
}

// Create creates a new user in the database
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (
			id, email, password_hash, name, role,
			deleted_at, deleted_by, is_anonymized,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10
		)
	`

	q := GetQueryable(ctx, r.pool)
	_, err := q.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.DeletedAt,
		user.DeletedBy,
		user.IsAnonymized,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation (duplicate email)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrCodeUniqueViolation {
			return apperrors.Conflict("user with this email already exists")
		}
		return apperrors.Wrapf(err, "failed to create user")
	}

	r.logger.WithContext(ctx).Info("user created",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("role", string(user.Role)),
	)

	return nil
}

// FindByID retrieves a user by their unique ID
// IMPORTANT: Excludes password_hash for security
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := `
		SELECT
			id, email, name, role,
			deleted_at, deleted_by, is_anonymized,
			created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user entity.User
	q := GetQueryable(ctx, r.pool)
	err := q.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.DeletedAt,
		&user.DeletedBy,
		&user.IsAnonymized,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Wrapf(err, "failed to find user by id")
	}

	return &user, nil
}

// FindByEmail retrieves a user by their email address
// IMPORTANT: Excludes password_hash for security
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT
			id, email, name, role,
			deleted_at, deleted_by, is_anonymized,
			created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user entity.User
	q := GetQueryable(ctx, r.pool)
	err := q.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.DeletedAt,
		&user.DeletedBy,
		&user.IsAnonymized,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Wrapf(err, "failed to find user by email")
	}

	return &user, nil
}

// FindByEmailWithPassword retrieves a user by email including password hash
// This should ONLY be used for authentication purposes
func (r *UserRepository) FindByEmailWithPassword(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT
			id, email, password_hash, name, role,
			deleted_at, deleted_by, is_anonymized,
			created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user entity.User
	q := GetQueryable(ctx, r.pool)
	err := q.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.DeletedAt,
		&user.DeletedBy,
		&user.IsAnonymized,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Wrapf(err, "failed to find user by email with password")
	}

	return &user, nil
}

// Update updates an existing user's information
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE users
		SET
			email = $2,
			password_hash = $3,
			name = $4,
			role = $5,
			deleted_at = $6,
			deleted_by = $7,
			is_anonymized = $8,
			updated_at = $9
		WHERE id = $1
	`

	q := GetQueryable(ctx, r.pool)
	commandTag, err := q.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.DeletedAt,
		user.DeletedBy,
		user.IsAnonymized,
		user.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrCodeUniqueViolation {
			return apperrors.Conflict("user with this email already exists")
		}
		return apperrors.Wrapf(err, "failed to update user")
	}

	if commandTag.RowsAffected() == 0 {
		return apperrors.NotFound("user not found")
	}

	r.logger.WithContext(ctx).Info("user updated",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	return nil
}

// List retrieves a paginated list of users
// Excludes soft-deleted users and password_hash for security
func (r *UserRepository) List(ctx context.Context, offset, limit int) ([]*entity.User, int64, error) {
	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE deleted_at IS NULL
	`

	var total int64
	q := GetQueryable(ctx, r.pool)
	err := q.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, apperrors.Wrapf(err, "failed to count users")
	}

	// Get paginated results
	query := `
		SELECT
			id, email, name, role,
			deleted_at, deleted_by, is_anonymized,
			created_at, updated_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := q.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Wrapf(err, "failed to list users")
	}
	defer rows.Close()

	users := make([]*entity.User, 0, limit)
	for rows.Next() {
		var user entity.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.Role,
			&user.DeletedAt,
			&user.DeletedBy,
			&user.IsAnonymized,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, apperrors.Wrapf(err, "failed to scan user row")
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.Wrapf(err, "error iterating user rows")
	}

	return users, total, nil
}

// SoftDelete marks a user as deleted and anonymizes their PII
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	anonymizedEmail := fmt.Sprintf("deleted_%s@anonymized.local", id.String())
	anonymizedName := "Deleted User"

	query := `
		UPDATE users
		SET
			email = $2,
			name = $3,
			deleted_at = $4,
			deleted_by = $5,
			is_anonymized = $6,
			updated_at = $7
		WHERE id = $1 AND deleted_at IS NULL
	`

	q := GetQueryable(ctx, r.pool)
	commandTag, err := q.Exec(ctx, query,
		id,
		anonymizedEmail,
		anonymizedName,
		now,
		deletedBy,
		true,
		now,
	)
	if err != nil {
		return apperrors.Wrapf(err, "failed to soft delete user")
	}

	if commandTag.RowsAffected() == 0 {
		return apperrors.NotFound("user not found or already deleted")
	}

	r.logger.WithContext(ctx).Info("user soft deleted and anonymized",
		zap.String("user_id", id.String()),
		zap.String("deleted_by", deletedBy.String()),
	)

	return nil
}

// ExistsByEmail checks if a user with the given email exists
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	q := GetQueryable(ctx, r.pool)
	err := q.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, apperrors.Wrapf(err, "failed to check if user exists by email")
	}

	return exists, nil
}

// HealthCheck verifies the repository's database connection
func (r *UserRepository) HealthCheck(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Compile-time interface compliance check
var _ repository.UserRepository = (*UserRepository)(nil)
