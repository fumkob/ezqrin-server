package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gin-gonic/gin"

	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/response"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
)

// newTestContext creates a gin.Context backed by an httptest.ResponseRecorder.
// The context has a minimal *http.Request with the given URL path set so that
// c.Request.URL.Path is usable inside response helpers (Instance field).
func newTestContext(path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	c.Request = req
	return c, w
}

var _ = Describe("Response helpers", func() {
	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
	})

	// -------------------------------------------------------------------------
	// Data
	// -------------------------------------------------------------------------
	Describe("Data", func() {
		When("called with a status code and payload", func() {
			Context("with a simple struct payload", func() {
				It("returns the correct status code and JSON body without a wrapper", func() {
					type payload struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					}

					c, w := newTestContext("/test")
					response.Data(c, http.StatusOK, payload{ID: 1, Name: "alice"})

					Expect(w.Code).To(Equal(http.StatusOK))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["id"]).To(BeEquivalentTo(1))
					Expect(result["name"]).To(Equal("alice"))

					// No extra "data" wrapper
					_, hasData := result["data"]
					Expect(hasData).To(BeFalse())
				})

				It("returns 201 Created when that status is requested", func() {
					c, w := newTestContext("/items")
					response.Data(c, http.StatusCreated, map[string]string{"key": "val"})

					Expect(w.Code).To(Equal(http.StatusCreated))
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// List
	// -------------------------------------------------------------------------
	Describe("List", func() {
		When("called with a slice and pagination meta", func() {
			Context("with a non-empty slice and valid meta", func() {
				It("returns the correct status code and a body with data and meta fields", func() {
					items := []string{"a", "b", "c"}
					meta := generated.PaginationMeta{
						Page:       2,
						PerPage:    10,
						Total:      25,
						TotalPages: 3,
					}

					c, w := newTestContext("/events")
					response.List(c, http.StatusOK, items, meta)

					Expect(w.Code).To(Equal(http.StatusOK))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())

					_, hasData := result["data"]
					Expect(hasData).To(BeTrue())

					metaRaw, hasMeta := result["meta"]
					Expect(hasMeta).To(BeTrue())

					metaMap, ok := metaRaw.(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(metaMap["page"]).To(BeEquivalentTo(2))
					Expect(metaMap["per_page"]).To(BeEquivalentTo(10))
					Expect(metaMap["total"]).To(BeEquivalentTo(25))
					Expect(metaMap["total_pages"]).To(BeEquivalentTo(3))
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// NoContent
	// -------------------------------------------------------------------------
	Describe("NoContent", func() {
		When("called", func() {
			Context("unconditionally", func() {
				It("returns 204 with an empty body", func() {
					c, w := newTestContext("/noop")
					response.NoContent(c)

					// Gin defers writing the HTTP status header until something is
					// written to the body.  For a 204 response there is no body, so
					// we flush the header explicitly to make the recorder reflect the
					// correct status code, then assert on both the recorder and the
					// gin writer's own status.
					c.Writer.WriteHeaderNow()
					Expect(w.Code).To(Equal(http.StatusNoContent))
					Expect(w.Body.Bytes()).To(BeEmpty())
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// Problem
	// -------------------------------------------------------------------------
	Describe("Problem", func() {
		When("called with explicit type, title and detail", func() {
			Context("with a 422 status", func() {
				It("returns the correct status, Content-Type header and RFC 9457 body fields", func() {
					c, w := newTestContext("/things/123")
					response.Problem(
						c,
						http.StatusUnprocessableEntity,
						"https://example.com/problems/unprocessable",
						"Unprocessable Entity",
						"The submitted data could not be processed",
					)

					Expect(w.Code).To(Equal(http.StatusUnprocessableEntity))
					Expect(w.Header().Get("Content-Type")).To(Equal("application/problem+json"))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["type"]).To(Equal("https://example.com/problems/unprocessable"))
					Expect(result["title"]).To(Equal("Unprocessable Entity"))
					Expect(result["status"]).To(BeEquivalentTo(422))
					Expect(result["detail"]).To(Equal("The submitted data could not be processed"))
					Expect(result["instance"]).To(Equal("/things/123"))
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// ProblemWithCode
	// -------------------------------------------------------------------------
	Describe("ProblemWithCode", func() {
		When("called with an application error code", func() {
			Context("with CodeNotFound", func() {
				It("returns 404, the correct code field and a type URL derived from the code", func() {
					c, w := newTestContext("/resources/99")
					response.ProblemWithCode(c, http.StatusNotFound, apperrors.CodeNotFound, "resource not found")

					Expect(w.Code).To(Equal(http.StatusNotFound))
					Expect(w.Header().Get("Content-Type")).To(Equal("application/problem+json"))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeNotFound))
					Expect(result["type"]).To(Equal(apperrors.ToTypeURL(apperrors.CodeNotFound)))
					Expect(result["title"]).To(Equal(apperrors.GetTitle(apperrors.CodeNotFound)))
					Expect(result["detail"]).To(Equal("resource not found"))
					Expect(result["instance"]).To(Equal("/resources/99"))
				})
			})

			Context("with CodeInternal", func() {
				It("returns 500 with the INTERNAL_ERROR code in the body", func() {
					c, w := newTestContext("/crash")
					response.ProblemWithCode(c, http.StatusInternalServerError, apperrors.CodeInternal, "something went wrong")

					Expect(w.Code).To(Equal(http.StatusInternalServerError))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeInternal))
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// ProblemFromError
	// -------------------------------------------------------------------------
	Describe("ProblemFromError", func() {
		When("the error is nil", func() {
			Context("unconditionally", func() {
				It("returns 204 No Content with an empty body", func() {
					c, w := newTestContext("/ok")
					response.ProblemFromError(c, nil)

					// Same as NoContent: flush the header so the recorder reflects 204.
					c.Writer.WriteHeaderNow()
					Expect(w.Code).To(Equal(http.StatusNoContent))
					Expect(w.Body.Bytes()).To(BeEmpty())
				})
			})
		})

		When("the error is an AppError of type Unauthorized", func() {
			Context("without validation errors", func() {
				It("returns 401 with the UNAUTHORIZED code and the AppError message", func() {
					appErr := apperrors.Unauthorized("token expired")

					c, w := newTestContext("/secure")
					response.ProblemFromError(c, appErr)

					Expect(w.Code).To(Equal(http.StatusUnauthorized))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeUnauthorized))
					Expect(result["detail"]).To(Equal("token expired"))
					Expect(result["status"]).To(BeEquivalentTo(http.StatusUnauthorized))
				})
			})
		})

		When("the error is an AppError of type NotFound", func() {
			Context("without validation errors", func() {
				It("returns 404 with the NOT_FOUND code", func() {
					appErr := apperrors.NotFound("user not found")

					c, w := newTestContext("/users/42")
					response.ProblemFromError(c, appErr)

					Expect(w.Code).To(Equal(http.StatusNotFound))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeNotFound))
					Expect(result["detail"]).To(Equal("user not found"))
				})
			})
		})

		When("the error is an AppError of type Validation", func() {
			Context("without field-level validation errors", func() {
				It("returns 400 with the VALIDATION_ERROR code", func() {
					appErr := apperrors.Validation("input is invalid")

					c, w := newTestContext("/submit")
					response.ProblemFromError(c, appErr)

					Expect(w.Code).To(Equal(http.StatusBadRequest))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeValidation))
					Expect(result["detail"]).To(Equal("input is invalid"))
				})
			})

			Context("with field-level validation errors attached", func() {
				It("returns 400 and includes an errors array in the body", func() {
					validationErrs := []generated.ValidationError{
						{Field: "email", Message: "must be a valid email"},
						{Field: "name", Message: "must not be blank"},
					}
					appErr := apperrors.Validation("input is invalid").
						WithValidationErrors(validationErrs)

					c, w := newTestContext("/submit")
					response.ProblemFromError(c, appErr)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Header().Get("Content-Type")).To(Equal("application/problem+json"))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeValidation))

					errsRaw, hasErrors := result["errors"]
					Expect(hasErrors).To(BeTrue())

					errsSlice, ok := errsRaw.([]interface{})
					Expect(ok).To(BeTrue())
					Expect(errsSlice).To(HaveLen(2))

					firstErr, ok := errsSlice[0].(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(firstErr["field"]).To(Equal("email"))
					Expect(firstErr["message"]).To(Equal("must be a valid email"))
				})
			})
		})

		When("the error is a plain (non-AppError) error", func() {
			Context("unconditionally", func() {
				It("returns 500 with the INTERNAL_ERROR code", func() {
					plainErr := errors.New("unexpected database failure")

					c, w := newTestContext("/ops")
					response.ProblemFromError(c, plainErr)

					Expect(w.Code).To(Equal(http.StatusInternalServerError))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeInternal))
				})
			})
		})

		When("the error is a wrapped AppError", func() {
			Context("wrapped with fmt.Errorf %w", func() {
				It("unwraps correctly and returns the AppError status and code", func() {
					inner := apperrors.Forbidden("access denied")
					wrapped := apperrors.Wrap(inner, "handler layer")

					c, w := newTestContext("/admin")
					response.ProblemFromError(c, wrapped)

					Expect(w.Code).To(Equal(http.StatusForbidden))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeForbidden))
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// ValidationProblem
	// -------------------------------------------------------------------------
	Describe("ValidationProblem", func() {
		When("called with a list of validation errors", func() {
			Context("with two field errors", func() {
				It("returns 400, Content-Type application/problem+json, and the errors array", func() {
					validationErrs := []generated.ValidationError{
						{Field: "start_date", Message: "must be in the future"},
						{Field: "capacity", Message: "must be greater than zero"},
					}

					c, w := newTestContext("/events")
					response.ValidationProblem(c, validationErrs)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Header().Get("Content-Type")).To(Equal("application/problem+json"))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeValidation))
					Expect(result["type"]).To(Equal(apperrors.ToTypeURL(apperrors.CodeValidation)))
					Expect(result["status"]).To(BeEquivalentTo(http.StatusBadRequest))

					errsRaw, hasErrors := result["errors"]
					Expect(hasErrors).To(BeTrue())

					errsSlice, ok := errsRaw.([]interface{})
					Expect(ok).To(BeTrue())
					Expect(errsSlice).To(HaveLen(2))
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// InternalProblem
	// -------------------------------------------------------------------------
	Describe("InternalProblem", func() {
		When("called with a detail message", func() {
			Context("unconditionally", func() {
				It("returns 500 with the INTERNAL_ERROR code", func() {
					c, w := newTestContext("/crash")
					response.InternalProblem(c, "database unreachable")

					Expect(w.Code).To(Equal(http.StatusInternalServerError))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeInternal))
					Expect(result["detail"]).To(Equal("database unreachable"))
					Expect(result["status"]).To(BeEquivalentTo(http.StatusInternalServerError))
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// NotFoundProblem
	// -------------------------------------------------------------------------
	Describe("NotFoundProblem", func() {
		When("called with a detail message", func() {
			Context("unconditionally", func() {
				It("returns 404 with the NOT_FOUND code", func() {
					c, w := newTestContext("/items/99")
					response.NotFoundProblem(c, "item 99 does not exist")

					Expect(w.Code).To(Equal(http.StatusNotFound))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeNotFound))
					Expect(result["detail"]).To(Equal("item 99 does not exist"))
					Expect(result["status"]).To(BeEquivalentTo(http.StatusNotFound))
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// UnauthorizedProblem
	// -------------------------------------------------------------------------
	Describe("UnauthorizedProblem", func() {
		When("called with a detail message", func() {
			Context("unconditionally", func() {
				It("returns 401 with the UNAUTHORIZED code", func() {
					c, w := newTestContext("/profile")
					response.UnauthorizedProblem(c, "missing or invalid token")

					Expect(w.Code).To(Equal(http.StatusUnauthorized))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeUnauthorized))
					Expect(result["detail"]).To(Equal("missing or invalid token"))
					Expect(result["status"]).To(BeEquivalentTo(http.StatusUnauthorized))
				})
			})
		})
	})

	// -------------------------------------------------------------------------
	// ForbiddenProblem
	// -------------------------------------------------------------------------
	Describe("ForbiddenProblem", func() {
		When("called with a detail message", func() {
			Context("unconditionally", func() {
				It("returns 403 with the FORBIDDEN code", func() {
					c, w := newTestContext("/admin/settings")
					response.ForbiddenProblem(c, "you do not have permission")

					Expect(w.Code).To(Equal(http.StatusForbidden))

					var result map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
					Expect(result["code"]).To(Equal(apperrors.CodeForbidden))
					Expect(result["detail"]).To(Equal("you do not have permission"))
					Expect(result["status"]).To(BeEquivalentTo(http.StatusForbidden))
				})
			})
		})
	})
})
