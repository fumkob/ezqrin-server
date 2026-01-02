package database

import (
	"context"
	"fmt"

	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// txKey is the context key for storing pgx.Tx
type txKey struct{}

// WithTransaction executes the provided function within a database transaction.
// It automatically handles begin, commit, and rollback based on the function's return value.
// If the function returns an error, the transaction is rolled back; otherwise, it is committed.
//
// Example usage:
//
//	err := database.WithTransaction(ctx, pool, func(ctx context.Context) error {
//		// Perform database operations using ctx
//		_, err := repo.CreateUser(ctx, user)
//		if err != nil {
//			return err // Transaction will be rolled back
//		}
//		_, err = repo.CreateProfile(ctx, profile)
//		return err // Transaction will be committed if no error
//	})
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context) error) error {
	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return apperrors.Wrapf(err, "failed to begin transaction")
	}

	// Store transaction in context
	txCtx := context.WithValue(ctx, txKey{}, tx)

	// Execute function and handle commit/rollback
	defer func() {
		if p := recover(); p != nil {
			// Rollback on panic using background context (original context may be cancelled)
			_ = tx.Rollback(context.Background())
			panic(p) // Re-throw panic after rollback
		}
	}()

	// Execute the provided function
	if err := fn(txCtx); err != nil {
		// Rollback on error using background context (original context may be cancelled)
		if rbErr := tx.Rollback(context.Background()); rbErr != nil {
			// Wrap both errors properly to preserve error context
			return fmt.Errorf("transaction failed: %w (rollback also failed: %w)", err, rbErr)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return apperrors.Wrapf(err, "failed to commit transaction")
	}

	return nil
}

// GetTx retrieves the pgx.Tx from context if present, otherwise returns nil.
// This allows repository methods to use the transaction if one is active.
func GetTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}

// GetQueryable returns either the transaction from context or the pool.
// This ensures repository methods work both inside and outside transactions.
//
// Example usage in repository:
//
//	func (r *userRepo) Create(ctx context.Context, user *entity.User) error {
//		q := database.GetQueryable(ctx, r.pool)
//		_, err := q.Exec(ctx, "INSERT INTO users ...", user.ID, user.Email)
//		return err
//	}
func GetQueryable(ctx context.Context, pool *pgxpool.Pool) Queryable {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return pool
}

// Queryable is an interface that abstracts common database operations.
// Both pgxpool.Pool and pgx.Tx implement this interface, allowing
// repository methods to work seamlessly with or without transactions.
type Queryable interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// Verify that pgxpool.Pool and pgx.Tx implement Queryable interface at compile time
var (
	_ Queryable = (*pgxpool.Pool)(nil)
	_ Queryable = (pgx.Tx)(nil)
)

// TxFunc is a function type that represents a database operation within a transaction
type TxFunc func(context.Context) error

// WithTxFunc is an alias for WithTransaction for better readability in some contexts
func WithTxFunc(ctx context.Context, pool *pgxpool.Pool, fn TxFunc) error {
	return WithTransaction(ctx, pool, fn)
}

// MustBeginTx starts a new transaction and panics if it fails.
// This is useful for test setup where transaction failure is unacceptable.
// DO NOT use in production code; use WithTransaction instead.
func MustBeginTx(ctx context.Context, pool *pgxpool.Pool) pgx.Tx {
	tx, err := pool.Begin(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to begin transaction: %v", err))
	}
	return tx
}
