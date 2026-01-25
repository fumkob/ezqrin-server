package errors_test

import (
	"errors"
	"net/http"

	pkgerrors "github.com/fumkob/ezqrin-server/pkg/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppError", func() {
	When("creating error instances", func() {
		Context("with NotFound constructor", func() {
			It("should create error with correct properties", func() {
				err := pkgerrors.NotFound("resource not found")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeNotFound))
				Expect(err.Message).To(Equal("resource not found"))
				Expect(err.StatusCode).To(Equal(http.StatusNotFound))
				Expect(err.Err).To(BeNil())
			})

			It("should implement error interface", func() {
				err := pkgerrors.NotFound("user not found")

				Expect(err.Error()).To(ContainSubstring("NOT_FOUND"))
				Expect(err.Error()).To(ContainSubstring("user not found"))
			})
		})

		Context("with NotFoundf constructor", func() {
			It("should create formatted error", func() {
				err := pkgerrors.NotFoundf("user %s not found", "user-123")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeNotFound))
				Expect(err.Message).To(Equal("user user-123 not found"))
				Expect(err.StatusCode).To(Equal(http.StatusNotFound))
			})

			It("should format with multiple arguments", func() {
				err := pkgerrors.NotFoundf("resource %s with id %d not found", "event", 42)

				Expect(err.Message).To(Equal("resource event with id 42 not found"))
			})
		})

		Context("with Validation constructor", func() {
			It("should create validation error", func() {
				err := pkgerrors.Validation("invalid email format")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeValidation))
				Expect(err.Message).To(Equal("invalid email format"))
				Expect(err.StatusCode).To(Equal(http.StatusBadRequest))
			})
		})

		Context("with Validationf constructor", func() {
			It("should create formatted validation error", func() {
				err := pkgerrors.Validationf("field %s is required", "email")

				Expect(err.Code).To(Equal(pkgerrors.CodeValidation))
				Expect(err.Message).To(Equal("field email is required"))
			})
		})

		Context("with Unauthorized constructor", func() {
			It("should create unauthorized error", func() {
				err := pkgerrors.Unauthorized("invalid credentials")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeUnauthorized))
				Expect(err.Message).To(Equal("invalid credentials"))
				Expect(err.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("with Unauthorizedf constructor", func() {
			It("should create formatted unauthorized error", func() {
				err := pkgerrors.Unauthorizedf("token %s is invalid", "abc123")

				Expect(err.Message).To(Equal("token abc123 is invalid"))
			})
		})

		Context("with Forbidden constructor", func() {
			It("should create forbidden error", func() {
				err := pkgerrors.Forbidden("access denied")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeForbidden))
				Expect(err.Message).To(Equal("access denied"))
				Expect(err.StatusCode).To(Equal(http.StatusForbidden))
			})
		})

		Context("with Forbiddenf constructor", func() {
			It("should create formatted forbidden error", func() {
				err := pkgerrors.Forbiddenf("user %s cannot access resource %s", "user-1", "event-2")

				Expect(err.Message).To(Equal("user user-1 cannot access resource event-2"))
			})
		})

		Context("with Internal constructor", func() {
			It("should create internal server error", func() {
				err := pkgerrors.Internal("database connection failed")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeInternal))
				Expect(err.Message).To(Equal("database connection failed"))
				Expect(err.StatusCode).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("with Internalf constructor", func() {
			It("should create formatted internal error", func() {
				err := pkgerrors.Internalf("failed to connect to %s", "redis")

				Expect(err.Message).To(Equal("failed to connect to redis"))
			})
		})

		Context("with Conflict constructor", func() {
			It("should create conflict error", func() {
				err := pkgerrors.Conflict("email already exists")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeConflict))
				Expect(err.Message).To(Equal("email already exists"))
				Expect(err.StatusCode).To(Equal(http.StatusConflict))
			})
		})

		Context("with Conflictf constructor", func() {
			It("should create formatted conflict error", func() {
				err := pkgerrors.Conflictf("user with email %s already exists", "test@example.com")

				Expect(err.Message).To(Equal("user with email test@example.com already exists"))
			})
		})

		Context("with BadRequest constructor", func() {
			It("should create bad request error", func() {
				err := pkgerrors.BadRequest("invalid request body")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeBadRequest))
				Expect(err.Message).To(Equal("invalid request body"))
				Expect(err.StatusCode).To(Equal(http.StatusBadRequest))
			})
		})

		Context("with BadRequestf constructor", func() {
			It("should create formatted bad request error", func() {
				err := pkgerrors.BadRequestf("invalid value %s for field %s", "abc", "age")

				Expect(err.Message).To(Equal("invalid value abc for field age"))
			})
		})

		Context("with TooManyRequests constructor", func() {
			It("should create too many requests error", func() {
				err := pkgerrors.TooManyRequests("rate limit exceeded")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeTooManyRequests))
				Expect(err.Message).To(Equal("rate limit exceeded"))
				Expect(err.StatusCode).To(Equal(http.StatusTooManyRequests))
			})
		})

		Context("with ServiceUnavailable constructor", func() {
			It("should create service unavailable error", func() {
				err := pkgerrors.ServiceUnavailable("service temporarily unavailable")

				Expect(err).NotTo(BeNil())
				Expect(err.Code).To(Equal(pkgerrors.CodeServiceUnavailable))
				Expect(err.Message).To(Equal("service temporarily unavailable"))
				Expect(err.StatusCode).To(Equal(http.StatusServiceUnavailable))
			})
		})
	})

	When("formatting error messages", func() {
		Context("with wrapped error", func() {
			It("should include wrapped error in message", func() {
				originalErr := errors.New("connection timeout")
				appErr := pkgerrors.WrapAppError(
					pkgerrors.Internal("database error"),
					originalErr,
				)

				Expect(appErr.Error()).To(ContainSubstring("INTERNAL_ERROR"))
				Expect(appErr.Error()).To(ContainSubstring("database error"))
				Expect(appErr.Error()).To(ContainSubstring("connection timeout"))
			})
		})

		Context("without wrapped error", func() {
			It("should only include code and message", func() {
				err := pkgerrors.NotFound("resource not found")

				errorMsg := err.Error()
				Expect(errorMsg).To(ContainSubstring("NOT_FOUND"))
				Expect(errorMsg).To(ContainSubstring("resource not found"))
			})
		})
	})

	When("wrapping errors", func() {
		Context("with Wrap function", func() {
			It("should wrap error with context", func() {
				originalErr := errors.New("database error")
				wrappedErr := pkgerrors.Wrap(originalErr, "failed to save user")

				Expect(wrappedErr).NotTo(BeNil())
				Expect(wrappedErr.Error()).To(ContainSubstring("failed to save user"))
				Expect(wrappedErr.Error()).To(ContainSubstring("database error"))
				Expect(errors.Is(wrappedErr, originalErr)).To(BeTrue())
			})

			It("should return nil when wrapping nil error", func() {
				wrappedErr := pkgerrors.Wrap(nil, "some context")

				Expect(wrappedErr).To(BeNil())
			})
		})

		Context("with Wrapf function", func() {
			It("should wrap error with formatted context", func() {
				originalErr := errors.New("permission denied")
				wrappedErr := pkgerrors.Wrapf(originalErr, "user %s cannot %s", "user-123", "delete event")

				Expect(wrappedErr).NotTo(BeNil())
				Expect(wrappedErr.Error()).To(ContainSubstring("user user-123 cannot delete event"))
				Expect(wrappedErr.Error()).To(ContainSubstring("permission denied"))
			})

			It("should return nil when wrapping nil error", func() {
				wrappedErr := pkgerrors.Wrapf(nil, "context with %s", "formatting")

				Expect(wrappedErr).To(BeNil())
			})
		})

		Context("with WrapAppError function", func() {
			It("should wrap underlying error into AppError", func() {
				originalErr := errors.New("network timeout")
				appErr := pkgerrors.Internal("operation failed")
				wrappedErr := pkgerrors.WrapAppError(appErr, originalErr)

				Expect(wrappedErr).NotTo(BeNil())
				Expect(wrappedErr.Err).To(Equal(originalErr))
				Expect(wrappedErr.Code).To(Equal(pkgerrors.CodeInternal))
			})

			It("should handle nil AppError", func() {
				originalErr := errors.New("some error")
				wrappedErr := pkgerrors.WrapAppError(nil, originalErr)

				Expect(wrappedErr).NotTo(BeNil())
				Expect(wrappedErr.Code).To(Equal(pkgerrors.CodeInternal))
				Expect(wrappedErr.Message).To(ContainSubstring("unexpected nil app error"))
			})
		})
	})

	When("unwrapping errors", func() {
		Context("with Unwrap method", func() {
			It("should return wrapped error", func() {
				originalErr := errors.New("root cause")
				appErr := pkgerrors.WrapAppError(
					pkgerrors.NotFound("resource missing"),
					originalErr,
				)

				unwrappedErr := appErr.Unwrap()

				Expect(unwrappedErr).To(Equal(originalErr))
			})

			It("should return nil when no wrapped error", func() {
				appErr := pkgerrors.Validation("invalid input")

				unwrappedErr := appErr.Unwrap()

				Expect(unwrappedErr).To(BeNil())
			})
		})

		Context("with errors.Is", func() {
			It("should work with wrapped AppError", func() {
				originalErr := errors.New("specific error")
				appErr := pkgerrors.WrapAppError(
					pkgerrors.Internal("wrapper"),
					originalErr,
				)

				Expect(errors.Is(appErr, originalErr)).To(BeTrue())
			})

			It("should not match different errors", func() {
				err1 := errors.New("error one")
				err2 := errors.New("error two")
				appErr := pkgerrors.WrapAppError(pkgerrors.Internal("wrapper"), err1)

				Expect(errors.Is(appErr, err2)).To(BeFalse())
			})
		})

		Context("with errors.As", func() {
			It("should extract AppError from wrapped error", func() {
				appErr := pkgerrors.NotFound("not found")
				wrappedErr := pkgerrors.Wrap(appErr, "additional context")

				var extractedAppErr *pkgerrors.AppError
				found := errors.As(wrappedErr, &extractedAppErr)

				Expect(found).To(BeTrue())
				Expect(extractedAppErr.Code).To(Equal(pkgerrors.CodeNotFound))
			})
		})
	})

	When("extracting status codes", func() {
		Context("with GetStatusCode function", func() {
			It("should return correct status for NotFound", func() {
				err := pkgerrors.NotFound("not found")

				statusCode := pkgerrors.GetStatusCode(err)

				Expect(statusCode).To(Equal(http.StatusNotFound))
			})

			It("should return correct status for Validation", func() {
				err := pkgerrors.Validation("invalid")

				statusCode := pkgerrors.GetStatusCode(err)

				Expect(statusCode).To(Equal(http.StatusBadRequest))
			})

			It("should return correct status for Unauthorized", func() {
				err := pkgerrors.Unauthorized("unauthorized")

				statusCode := pkgerrors.GetStatusCode(err)

				Expect(statusCode).To(Equal(http.StatusUnauthorized))
			})

			It("should return correct status for Forbidden", func() {
				err := pkgerrors.Forbidden("forbidden")

				statusCode := pkgerrors.GetStatusCode(err)

				Expect(statusCode).To(Equal(http.StatusForbidden))
			})

			It("should return correct status for Internal", func() {
				err := pkgerrors.Internal("internal error")

				statusCode := pkgerrors.GetStatusCode(err)

				Expect(statusCode).To(Equal(http.StatusInternalServerError))
			})

			It("should return correct status for Conflict", func() {
				err := pkgerrors.Conflict("conflict")

				statusCode := pkgerrors.GetStatusCode(err)

				Expect(statusCode).To(Equal(http.StatusConflict))
			})

			It("should return 500 for non-AppError", func() {
				err := errors.New("generic error")

				statusCode := pkgerrors.GetStatusCode(err)

				Expect(statusCode).To(Equal(http.StatusInternalServerError))
			})

			It("should return 200 for nil error", func() {
				statusCode := pkgerrors.GetStatusCode(nil)

				Expect(statusCode).To(Equal(http.StatusOK))
			})

			It("should extract status from wrapped error", func() {
				appErr := pkgerrors.NotFound("not found")
				wrappedErr := pkgerrors.Wrap(appErr, "context")

				statusCode := pkgerrors.GetStatusCode(wrappedErr)

				Expect(statusCode).To(Equal(http.StatusNotFound))
			})
		})
	})

	When("extracting error codes", func() {
		Context("with GetErrorCode function", func() {
			It("should return correct code for NotFound", func() {
				err := pkgerrors.NotFound("not found")

				code := pkgerrors.GetErrorCode(err)

				Expect(code).To(Equal(pkgerrors.CodeNotFound))
			})

			It("should return correct code for Validation", func() {
				err := pkgerrors.Validation("invalid")

				code := pkgerrors.GetErrorCode(err)

				Expect(code).To(Equal(pkgerrors.CodeValidation))
			})

			It("should return INTERNAL_ERROR for non-AppError", func() {
				err := errors.New("generic error")

				code := pkgerrors.GetErrorCode(err)

				Expect(code).To(Equal(pkgerrors.CodeInternal))
			})

			It("should return empty string for nil error", func() {
				code := pkgerrors.GetErrorCode(nil)

				Expect(code).To(Equal(""))
			})

			It("should extract code from wrapped error", func() {
				appErr := pkgerrors.Unauthorized("unauthorized")
				wrappedErr := pkgerrors.Wrap(appErr, "context")

				code := pkgerrors.GetErrorCode(wrappedErr)

				Expect(code).To(Equal(pkgerrors.CodeUnauthorized))
			})
		})
	})

	When("checking error types", func() {
		Context("with IsNotFound function", func() {
			It("should return true for NotFound error", func() {
				err := pkgerrors.NotFound("not found")

				Expect(pkgerrors.IsNotFound(err)).To(BeTrue())
			})

			It("should return true for wrapped NotFound error", func() {
				appErr := pkgerrors.NotFound("not found")
				wrappedErr := pkgerrors.Wrap(appErr, "context")

				Expect(pkgerrors.IsNotFound(wrappedErr)).To(BeTrue())
			})

			It("should return false for other error types", func() {
				err := pkgerrors.Validation("invalid")

				Expect(pkgerrors.IsNotFound(err)).To(BeFalse())
			})

			It("should return false for non-AppError", func() {
				err := errors.New("generic error")

				Expect(pkgerrors.IsNotFound(err)).To(BeFalse())
			})
		})

		Context("with IsValidation function", func() {
			It("should return true for Validation error", func() {
				err := pkgerrors.Validation("invalid")

				Expect(pkgerrors.IsValidation(err)).To(BeTrue())
			})

			It("should return false for other error types", func() {
				err := pkgerrors.NotFound("not found")

				Expect(pkgerrors.IsValidation(err)).To(BeFalse())
			})
		})

		Context("with IsUnauthorized function", func() {
			It("should return true for Unauthorized error", func() {
				err := pkgerrors.Unauthorized("unauthorized")

				Expect(pkgerrors.IsUnauthorized(err)).To(BeTrue())
			})

			It("should return false for other error types", func() {
				err := pkgerrors.Forbidden("forbidden")

				Expect(pkgerrors.IsUnauthorized(err)).To(BeFalse())
			})
		})

		Context("with IsForbidden function", func() {
			It("should return true for Forbidden error", func() {
				err := pkgerrors.Forbidden("forbidden")

				Expect(pkgerrors.IsForbidden(err)).To(BeTrue())
			})

			It("should return false for other error types", func() {
				err := pkgerrors.Unauthorized("unauthorized")

				Expect(pkgerrors.IsForbidden(err)).To(BeFalse())
			})
		})

		Context("with IsConflict function", func() {
			It("should return true for Conflict error", func() {
				err := pkgerrors.Conflict("conflict")

				Expect(pkgerrors.IsConflict(err)).To(BeTrue())
			})

			It("should return false for other error types", func() {
				err := pkgerrors.BadRequest("bad request")

				Expect(pkgerrors.IsConflict(err)).To(BeFalse())
			})
		})
	})

	When("handling multiple wrapping levels", func() {
		Context("with deeply nested errors", func() {
			It("should preserve error chain", func() {
				rootErr := errors.New("root cause")
				level1 := pkgerrors.Wrap(rootErr, "level 1")
				level2 := pkgerrors.Wrap(level1, "level 2")
				level3 := pkgerrors.Wrap(level2, "level 3")

				Expect(errors.Is(level3, rootErr)).To(BeTrue())
				Expect(level3.Error()).To(ContainSubstring("level 3"))
				Expect(level3.Error()).To(ContainSubstring("level 2"))
				Expect(level3.Error()).To(ContainSubstring("level 1"))
			})

			It("should extract AppError from nested wrapping", func() {
				appErr := pkgerrors.NotFound("not found")
				wrapped1 := pkgerrors.Wrap(appErr, "first wrap")
				wrapped2 := pkgerrors.Wrap(wrapped1, "second wrap")

				var extractedAppErr *pkgerrors.AppError
				found := errors.As(wrapped2, &extractedAppErr)

				Expect(found).To(BeTrue())
				Expect(extractedAppErr.Code).To(Equal(pkgerrors.CodeNotFound))
			})
		})
	})

	When("handling edge cases", func() {
		Context("with nil errors", func() {
			It("should handle nil in IsNotFound", func() {
				Expect(pkgerrors.IsNotFound(nil)).To(BeFalse())
			})

			It("should handle nil in IsValidation", func() {
				Expect(pkgerrors.IsValidation(nil)).To(BeFalse())
			})

			It("should handle nil in IsUnauthorized", func() {
				Expect(pkgerrors.IsUnauthorized(nil)).To(BeFalse())
			})

			It("should handle nil in IsForbidden", func() {
				Expect(pkgerrors.IsForbidden(nil)).To(BeFalse())
			})

			It("should handle nil in IsConflict", func() {
				Expect(pkgerrors.IsConflict(nil)).To(BeFalse())
			})
		})

		Context("with empty messages", func() {
			It("should create error with empty message", func() {
				err := pkgerrors.NotFound("")

				Expect(err).NotTo(BeNil())
				Expect(err.Message).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("NOT_FOUND"))
			})
		})
	})
})
