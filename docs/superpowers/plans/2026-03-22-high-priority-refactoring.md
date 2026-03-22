# High Priority Refactoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate code duplication in handlers and fix inconsistent error handling in ParticipantRepository to align with established codebase patterns.

**Architecture:** This is a pure refactoring effort — no new features, no behavior changes. Each task extracts duplicated code into shared utilities or aligns patterns with existing conventions (UserRepository for error handling, middleware for context helpers). All existing tests must continue to pass.

**Tech Stack:** Go 1.26.1, Gin, pgx/v5 (pgconn.PgError), Ginkgo/Gomega, go.uber.org/mock

---

## File Map

| Action | File | Responsibility |
|--------|------|---------------|
| Modify | `internal/interface/api/middleware/auth.go` | Export `ExtractBearerToken()` |
| Modify | `internal/interface/api/handler/event.go` | Remove `getUserID`/`getUserRole`, use middleware helpers |
| Modify | `internal/interface/api/handler/participant.go` | Remove `getUserID`/`getUserRole`, remove duplicate `convertEmailPtr`, use middleware helpers |
| Modify | `internal/interface/api/handler/checkin.go` | Remove `getUserID`/`getUserRole`, use middleware helpers |
| Modify | `internal/interface/api/handler/auth.go` | Remove `extractBearerToken`, use middleware export |
| Modify | `internal/infrastructure/database/participant_repository.go` | Use `pgconn.PgError` codes, replace `fmt.Errorf` with `apperrors.Wrapf`, add logger |
| Modify | `internal/infrastructure/container/container.go` | Pass logger to `NewParticipantRepository` |
| Modify | `internal/interface/api/middleware/auth_test.go` | Add tests for `ExtractBearerToken` |
| Modify | `internal/infrastructure/database/participant_repository_test.go` | Update constructor call with logger |

---

## Task 1: Export `ExtractBearerToken` from middleware

**Files:**
- Modify: `internal/interface/api/middleware/auth.go:207-221` (rename `extractBearerToken` → `ExtractBearerToken`)
- Modify: `internal/interface/api/middleware/auth_test.go` (add test for `ExtractBearerToken`)

### Steps

- [ ] **Step 1: Write failing test for `ExtractBearerToken`**

Add to `internal/interface/api/middleware/auth_test.go`:

```go
Describe("ExtractBearerToken", func() {
    var c *gin.Context

    BeforeEach(func() {
        gin.SetMode(gin.TestMode)
        w := httptest.NewRecorder()
        c, _ = gin.CreateTestContext(w)
        c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
    })

    When("Authorization header has valid Bearer token", func() {
        It("returns the token", func() {
            c.Request.Header.Set("Authorization", "Bearer test-token-123")
            token := ExtractBearerToken(c)
            Expect(token).To(Equal("test-token-123"))
        })
    })

    When("Authorization header is empty", func() {
        It("returns empty string", func() {
            token := ExtractBearerToken(c)
            Expect(token).To(BeEmpty())
        })
    })

    When("Authorization header has wrong format", func() {
        It("returns empty string", func() {
            c.Request.Header.Set("Authorization", "Basic abc123")
            token := ExtractBearerToken(c)
            Expect(token).To(BeEmpty())
        })
    })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/interface/api/middleware/ -run "ExtractBearerToken" -v`
Expected: Compilation error — `ExtractBearerToken` is unexported (`extractBearerToken`)

- [ ] **Step 3: Rename `extractBearerToken` → `ExtractBearerToken` in middleware**

In `internal/interface/api/middleware/auth.go`, rename the function at line 207:

```go
// ExtractBearerToken extracts the Bearer token from the Authorization header.
func ExtractBearerToken(c *gin.Context) string {
```

Update **both** internal call sites in the same file:
- `Authenticate` method at line 75: `token := extractBearerToken(c)` → `token := ExtractBearerToken(c)`
- `OptionalAuth` method at line 134: `token := extractBearerToken(c)` → `token := ExtractBearerToken(c)`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/interface/api/middleware/ -v`
Expected: All tests PASS including new `ExtractBearerToken` tests

- [ ] **Step 5: Commit**

```bash
git add internal/interface/api/middleware/auth.go internal/interface/api/middleware/auth_test.go
git commit -m "♻️ Export ExtractBearerToken from middleware package"
```

---

## Task 2: Remove duplicate `extractBearerToken` from handler/auth.go

**Files:**
- Modify: `internal/interface/api/handler/auth.go:17-19,186-200` (remove `authHeaderParts` constant and `extractBearerToken` function, use middleware)

### Steps

- [ ] **Step 1: Run existing auth handler tests as baseline**

Run: `go test ./internal/interface/api/handler/ -run "AuthHandler" -v`
Expected: All PASS

- [ ] **Step 2: Remove duplicate `extractBearerToken` and use middleware import**

In `internal/interface/api/handler/auth.go`:

1. Remove lines 17-19 (the `authHeaderParts` constant)
2. Remove lines 186-200 (the `extractBearerToken` function)
3. Replace the call site at line 129 (`accessToken := extractBearerToken(c)`) with `accessToken := middleware.ExtractBearerToken(c)`
4. Add import for middleware package if not already present

- [ ] **Step 3: Run all handler tests to verify no regression**

Run: `go test ./internal/interface/api/handler/ -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add internal/interface/api/handler/auth.go
git commit -m "♻️ Remove duplicate extractBearerToken from auth handler"
```

---

## Task 3: Remove duplicate `getUserID`/`getUserRole` from all handlers

**Files:**
- Modify: `internal/interface/api/handler/event.go:251-273` (remove helpers, use `middleware.GetUserID`/`middleware.GetUserRole`)
- Modify: `internal/interface/api/handler/participant.go:453-475` (remove helpers)
- Modify: `internal/interface/api/handler/checkin.go:134-156` (remove helpers)

### Steps

- [ ] **Step 1: Run all handler tests as baseline**

Run: `go test ./internal/interface/api/handler/ -v`
Expected: All PASS

- [ ] **Step 2: Update EventHandler — remove helpers and update call sites**

In `internal/interface/api/handler/event.go`:

1. Delete `getUserID()` method (lines 251-261)
2. Delete `getUserRole()` method (lines 263-273)
3. Replace all call sites. The middleware `GetUserID` returns `(uuid.UUID, bool)` — the `bool` can be safely discarded because the auth middleware has already validated the user before these handlers execute.

Call sites to update (6 total):
- Line 42-43 (`GetEvents`): `role := h.getUserRole(c)` / `userID := h.getUserID(c)`
- Line 110 (`PostEvents`): `userID := h.getUserID(c)`
- Lines 162-163 (`GetEventsId`): `role := h.getUserRole(c)` / `userID := h.getUserID(c)`
- Lines 189-190 (`PutEventsId`): `role := h.getUserRole(c)` / `userID := h.getUserID(c)`
- Lines 205-206 (`DeleteEventsId`): `role := h.getUserRole(c)` / `userID := h.getUserID(c)`
- Lines 221-222 (`GetEventsIdStats`): `role := h.getUserRole(c)` / `userID := h.getUserID(c)`

Replace pattern:
```go
// Before:
role := h.getUserRole(c)
userID := h.getUserID(c)

// After:
role := middleware.GetUserRole(c)
userID, _ := middleware.GetUserID(c)
```

- [ ] **Step 3: Update ParticipantHandler — remove helpers and update call sites**

In `internal/interface/api/handler/participant.go`:

1. Delete `getUserID()` method (lines 453-463)
2. Delete `getUserRole()` method (lines 465-475)
3. Replace all call sites with `middleware.GetUserID(c)` and `middleware.GetUserRole(c)`

- [ ] **Step 4: Update CheckinHandler — remove helpers and update call sites**

In `internal/interface/api/handler/checkin.go`:

1. Delete `getUserID()` method (lines 134-144)
2. Delete `getUserRole()` method (lines 146-156)
3. Replace all call sites with `middleware.GetUserID(c)` and `middleware.GetUserRole(c)`

- [ ] **Step 5: Run all tests to verify no regression**

Run: `go test ./internal/interface/api/... -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/interface/api/handler/event.go internal/interface/api/handler/participant.go internal/interface/api/handler/checkin.go
git commit -m "♻️ Remove duplicate getUserID/getUserRole from handlers, use middleware"
```

---

## Task 4: Remove duplicate `convertEmailPtr` in ParticipantHandler

**Files:**
- Modify: `internal/interface/api/handler/participant.go:539-554` (remove duplicate, keep one)

### Steps

- [ ] **Step 1: Run participant handler tests as baseline**

Run: `go test ./internal/interface/api/handler/ -run "Participant" -v`
Expected: All PASS

- [ ] **Step 2: Identify and remove duplicate function**

In `internal/interface/api/handler/participant.go`:

1. `convertEmailPtr()` (lines 539-545) and `convertEmailPtrToStringPtr()` (lines 547-554) have identical implementations
2. Keep `convertEmailPtr()` (the one with the cleaner name)
3. Delete `convertEmailPtrToStringPtr()` (lines 547-554)
4. Find and replace the call site at line 257 (`convertEmailPtrToStringPtr(...)` → `convertEmailPtr(...)`)

Verify with: `grep -n "convertEmailPtrToStringPtr" internal/interface/api/handler/participant.go`

- [ ] **Step 3: Run tests to verify no regression**

Run: `go test ./internal/interface/api/handler/ -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add internal/interface/api/handler/participant.go
git commit -m "♻️ Remove duplicate convertEmailPtrToStringPtr in participant handler"
```

---

## Task 5: Fix ParticipantRepository error handling — use `pgconn.PgError`

**Files:**
- Modify: `internal/infrastructure/database/participant_repository.go:60-69` (use pgconn error codes instead of string matching)

### Steps

- [ ] **Step 1: Run existing participant repository tests as baseline**

Run: `go test ./internal/infrastructure/database/ -v`
Expected: All PASS (unit tests only; integration tests require DB and `-tags=integration`)

- [ ] **Step 2: Replace string-based error detection with `pgconn.PgError`**

In `internal/infrastructure/database/participant_repository.go`:

1. Add import: `"github.com/jackc/pgx/v5/pgconn"`
2. Note: `"errors"` import already exists at line 5
3. The constants `pgErrCodeUniqueViolation` and `pgErrCodeForeignKeyViolation` are already defined in `user_repository.go` in the same `database` package — reuse them directly.

Replace lines 60-69 in the `Create` method:

Before (actual code):
```go
	if err != nil {
		// Check for unique constraint violations
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"unique_event_email\" (SQLSTATE 23505)" {
			return apperrors.Conflict("participant with this email already exists for this event")
		}
		if err.Error() == "ERROR: duplicate key value violates unique constraint "+
			"\"participants_qr_code_key\" (SQLSTATE 23505)" {
			return apperrors.Conflict("QR code already exists")
		}
		return fmt.Errorf("failed to insert participant: %w", err)
	}
```

After:
```go
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrCodeUniqueViolation {
			switch pgErr.ConstraintName {
			case "unique_event_email":
				return apperrors.Conflict("participant with this email already exists for this event")
			case "participants_qr_code_key":
				return apperrors.Conflict("QR code already exists")
			default:
				return apperrors.Conflict("participant already exists")
			}
		}
		return apperrors.Wrapf(err, "failed to insert participant")
	}
```

Note: Preserve the original error messages exactly (`"QR code already exists"`, not `"participant with this QR code already exists"`).

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/infrastructure/database/`
Expected: Compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/database/participant_repository.go
git commit -m "♻️ Use pgconn.PgError for constraint violation detection in ParticipantRepository.Create"
```

---

## Task 6: Replace all `fmt.Errorf` with `apperrors.Wrapf` in ParticipantRepository

**Files:**
- Modify: `internal/infrastructure/database/participant_repository.go` (all remaining `fmt.Errorf` call sites)

### Steps

- [ ] **Step 1: Identify all remaining `fmt.Errorf` usage**

After Task 5, line 69 is already converted. The remaining 15 occurrences are:

| Line | Current code |
|------|-------------|
| 29 | `return fmt.Errorf("invalid participant: %w", err)` |
| 84 | `return fmt.Errorf("invalid participant: %w", err)` |
| 128 | `return fmt.Errorf("failed to insert participant batch: %w", err)` |
| 153 | `return nil, fmt.Errorf("failed to find participant: %w", err)` |
| 255 | `return nil, fmt.Errorf("failed to find participant by QR code: %w", err)` |
| 283 | `return nil, fmt.Errorf("failed to find participant by employee ID: %w", err)` |
| 292 | `return fmt.Errorf("invalid participant: %w", err)` |
| 327 | `return fmt.Errorf("failed to update participant: %w", err)` |
| 346 | `return fmt.Errorf("failed to delete participant: %w", err)` |
| 423 | `return false, fmt.Errorf("failed to check participant existence: %w", err)` |
| 455 | `return nil, fmt.Errorf("failed to get payment stats: %w", err)` |
| 536 | `return nil, fmt.Errorf("failed to query participants: %w", err)` |
| 544 | `return nil, fmt.Errorf("failed to scan participant: %w", err)` |
| 550 | `return nil, fmt.Errorf("error iterating participants: %w", err)` |
| 568 | `return 0, fmt.Errorf("failed to count participants: %w", err)` |

- [ ] **Step 2: Replace all `fmt.Errorf` with `apperrors.Wrapf`**

For each occurrence, replace:
```go
return fmt.Errorf("failed to <action>: %w", err)
```
With:
```go
return apperrors.Wrapf(err, "failed to <action>")
```

Note: `apperrors.Wrapf(err, format, args...)` wraps the error internally — the format string does NOT include `%w`.

Also remove the `"fmt"` import (line 6) since it will no longer be used.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/infrastructure/database/`
Expected: Compiles without errors

- [ ] **Step 4: Run tests**

Run: `go test ./internal/infrastructure/database/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/database/participant_repository.go
git commit -m "♻️ Replace fmt.Errorf with apperrors.Wrapf in ParticipantRepository"
```

---

## Task 7: Add logger injection to ParticipantRepository

**Files:**
- Modify: `internal/infrastructure/database/participant_repository.go:17-24` (add logger field and constructor param)
- Modify: `internal/infrastructure/container/container.go` (pass logger to constructor)
- Modify: `internal/infrastructure/database/participant_repository_test.go` (update constructor call)

### Steps

- [ ] **Step 1: Add logger field and update constructor**

In `internal/infrastructure/database/participant_repository.go`:

The struct is `participantRepository` (unexported) and the constructor returns `repository.ParticipantRepository` (interface).

Before:
```go
type participantRepository struct {
	pool *pgxpool.Pool
}

func NewParticipantRepository(pool *pgxpool.Pool) repository.ParticipantRepository {
	return &participantRepository{pool: pool}
}
```

After:
```go
type participantRepository struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

func NewParticipantRepository(pool *pgxpool.Pool, logger *logger.Logger) repository.ParticipantRepository {
	return &participantRepository{pool: pool, logger: logger}
}
```

Add import: `"github.com/fumkob/ezqrin-server/pkg/logger"` (verify exact module path from `go.mod`).

- [ ] **Step 2: Update container to pass logger**

In `internal/infrastructure/container/container.go`, find where `NewParticipantRepository` is called and add the logger argument:

Before:
```go
participantRepo := database.NewParticipantRepository(db.GetPool())
```

After:
```go
participantRepo := database.NewParticipantRepository(db.GetPool(), logger)
```

- [ ] **Step 3: Update integration test constructor call**

In `internal/infrastructure/database/participant_repository_test.go` (build tag: `//go:build integration`), update the `BeforeEach`:

Before:
```go
repo = database.NewParticipantRepository(db.GetPool())
```

After:
```go
testLogger, _ := logger.NewLogger(&logger.Config{Level: "debug", Format: "console"})
repo = database.NewParticipantRepository(db.GetPool(), testLogger)
```

Add the logger import to the test file.

- [ ] **Step 4: Verify compilation**

Run: `go build ./...`
Expected: Compiles without errors

- [ ] **Step 5: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/infrastructure/database/participant_repository.go internal/infrastructure/container/container.go internal/infrastructure/database/participant_repository_test.go
git commit -m "♻️ Add logger injection to ParticipantRepository"
```

---

## Execution Order

Tasks must be executed in this order due to dependencies:

```
Task 1 (export ExtractBearerToken)
  → Task 2 (remove duplicate from auth handler) [depends on Task 1]
  → Task 3 (remove getUserID/getUserRole from handlers) [independent of Task 2, but same package]

Task 4 (remove duplicate convertEmailPtr) [independent]

Task 5 (pgconn.PgError in Create) [independent]
  → Task 6 (replace all fmt.Errorf) [depends on Task 5 — same file]
    → Task 7 (add logger injection) [depends on Task 6 — same file]
```

**Parallelizable groups:**
- Group A: Tasks 1 → 2 → 3
- Group B: Task 4
- Group C: Tasks 5 → 6 → 7

Groups A, B, and C can run in parallel if using separate worktrees.
