package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/handler"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/internal/usecase/participant"
	participantMocks "github.com/fumkob/ezqrin-server/internal/usecase/participant/mocks"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

// newQRCodeHandlerRouter creates a Gin test router with QRCodeHandler routes, injecting auth context.
func newQRCodeHandlerRouter(uc participant.Usecase, userID uuid.UUID, role string, log *logger.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Set(middleware.ContextKeyUserRole, role)
		c.Next()
	})

	h := handler.NewQRCodeHandler(uc, log)

	r.POST("/events/:id/qrcodes/send", func(c *gin.Context) {
		id, _ := uuid.Parse(c.Param("id"))
		h.SendEventQRCodes(c, generated.EventIDParam(id))
	})

	return r
}

var _ = Describe("QRCodeHandler", func() {
	var (
		log     *logger.Logger
		eventID uuid.UUID
		userID  uuid.UUID
		ctrl    *gomock.Controller
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		log = newTestLogger()
		eventID = uuid.New()
		userID = uuid.New()
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("SendEventQRCodes", func() {
		When("the request body is invalid", func() {
			Context("with malformed JSON", func() {
				It("should return 400 Bad Request", func() {
					mockUC := participantMocks.NewMockUsecase(ctrl)
					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(`{not valid json`),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
				})
			})

			Context("with missing Content-Type and no body", func() {
				It("should return 400 Bad Request", func() {
					mockUC := participantMocks.NewMockUsecase(ctrl)
					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(`not-json-at-all`),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
				})
			})
		})

		When("the usecase succeeds with all sends successful", func() {
			Context("with send_to_all=true", func() {
				It("should return 200 OK with SentCount and FailedCount=0", func() {
					var capturedInput participant.SendQRCodesInput

					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(
							_ context.Context, _ uuid.UUID, _ bool, input participant.SendQRCodesInput,
						) (participant.SendQRCodesOutput, error) {
							capturedInput = input
							return participant.SendQRCodesOutput{
								SentCount:   5,
								FailedCount: 0,
								Total:       5,
								Failures:    []participant.SendQRCodeFailure{},
							}, nil
						})

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					reqBody := `{"send_to_all": true}`
					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
					Expect(body["sent_count"]).To(BeEquivalentTo(5))
					Expect(body["failed_count"]).To(BeEquivalentTo(0))
					Expect(body["total"]).To(BeEquivalentTo(5))
					Expect(body["failures"]).To(BeEmpty())

					Expect(capturedInput.SendToAll).To(BeTrue())
					Expect(capturedInput.EventID).To(Equal(eventID))
				})
			})

			Context("with specific participant_ids", func() {
				It("should return 200 OK and correctly convert OpenAPI UUIDs to domain UUIDs", func() {
					pid1 := uuid.New()
					pid2 := uuid.New()
					var capturedInput participant.SendQRCodesInput

					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(
							_ context.Context, _ uuid.UUID, _ bool, input participant.SendQRCodesInput,
						) (participant.SendQRCodesOutput, error) {
							capturedInput = input
							return participant.SendQRCodesOutput{
								SentCount:   2,
								FailedCount: 0,
								Total:       2,
								Failures:    []participant.SendQRCodeFailure{},
							}, nil
						})

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					reqBody, err := json.Marshal(map[string]interface{}{
						"participant_ids": []string{pid1.String(), pid2.String()},
					})
					Expect(err).NotTo(HaveOccurred())

					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(string(reqBody)),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(capturedInput.ParticipantIDs).To(ConsistOf(pid1, pid2))
					Expect(capturedInput.SendToAll).To(BeFalse())
				})
			})
		})

		When("the usecase has partial failures", func() {
			Context("when some sends succeeded and some failed", func() {
				It("should return 207 Multi-Status with failure details", func() {
					failedParticipantID := uuid.New()
					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(participant.SendQRCodesOutput{
							SentCount:   3,
							FailedCount: 1,
							Total:       4,
							Failures: []participant.SendQRCodeFailure{
								{
									ParticipantID: failedParticipantID,
									Email:         "failed@example.com",
									Reason:        "SMTP connection refused",
								},
							},
						}, nil)

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					reqBody := `{"send_to_all": true}`
					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusMultiStatus))

					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
					Expect(body["sent_count"]).To(BeEquivalentTo(3))
					Expect(body["failed_count"]).To(BeEquivalentTo(1))
					Expect(body["total"]).To(BeEquivalentTo(4))

					failures, ok := body["failures"].([]interface{})
					Expect(ok).To(BeTrue())
					Expect(failures).To(HaveLen(1))

					failure := failures[0].(map[string]interface{})
					Expect(failure["participant_id"]).To(Equal(failedParticipantID.String()))
					Expect(failure["email"]).To(Equal("failed@example.com"))
					Expect(failure["reason"]).To(Equal("SMTP connection refused"))
				})
			})

			Context("when all sends failed (SentCount=0, FailedCount>0)", func() {
				It("should return 200 OK (not 207) because SentCount is not > 0", func() {
					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(participant.SendQRCodesOutput{
							SentCount:   0,
							FailedCount: 2,
							Total:       2,
							Failures: []participant.SendQRCodeFailure{
								{ParticipantID: uuid.New(), Email: "a@example.com", Reason: "timeout"},
								{ParticipantID: uuid.New(), Email: "b@example.com", Reason: "timeout"},
							},
						}, nil)

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					reqBody := `{"send_to_all": true}`
					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					// 207 is only returned when SentCount>0 AND FailedCount>0
					Expect(w.Code).To(Equal(http.StatusOK))

					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
					Expect(body["sent_count"]).To(BeEquivalentTo(0))
					Expect(body["failed_count"]).To(BeEquivalentTo(2))
				})
			})
		})

		When("the usecase returns an error", func() {
			Context("with a Forbidden error (caller lacks permission)", func() {
				It("should return 403 Forbidden", func() {
					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(participant.SendQRCodesOutput{}, apperrors.Forbidden("not the event organizer"))

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					reqBody := `{"send_to_all": true}`
					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusForbidden))
				})
			})

			Context("with a NotFound error (event does not exist)", func() {
				It("should return 404 Not Found", func() {
					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(participant.SendQRCodesOutput{}, apperrors.NotFound("event not found"))

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					reqBody := `{"send_to_all": true}`
					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNotFound))
				})
			})
		})

		When("the email_template field is specified", func() {
			Context("with template=detailed", func() {
				It("should pass the template value to the usecase", func() {
					var capturedInput participant.SendQRCodesInput

					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(
							_ context.Context, _ uuid.UUID, _ bool, input participant.SendQRCodesInput,
						) (participant.SendQRCodesOutput, error) {
							capturedInput = input
							return participant.SendQRCodesOutput{SentCount: 1, Total: 1}, nil
						})

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					detailedTemplate := generated.Detailed
					reqBody, err := json.Marshal(generated.SendQRCodesRequest{
						SendToAll:     boolPtr(true),
						EmailTemplate: &detailedTemplate,
					})
					Expect(err).NotTo(HaveOccurred())

					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(string(reqBody)),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(capturedInput.EmailTemplate).To(Equal("detailed"))
				})
			})

			Context("with no email_template specified", func() {
				It("should default to template=default", func() {
					var capturedInput participant.SendQRCodesInput

					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(
							_ context.Context, _ uuid.UUID, _ bool, input participant.SendQRCodesInput,
						) (participant.SendQRCodesOutput, error) {
							capturedInput = input
							return participant.SendQRCodesOutput{SentCount: 1, Total: 1}, nil
						})

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					reqBody := `{"send_to_all": true}`
					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(capturedInput.EmailTemplate).To(Equal("default"))
				})
			})
		})

		When("called as an admin user", func() {
			Context("when the admin sends QR codes for another organizer's event", func() {
				It("should pass isAdmin=true to the usecase", func() {
					adminID := uuid.New()
					var capturedIsAdmin bool

					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(
							_ context.Context, _ uuid.UUID, isAdmin bool, input participant.SendQRCodesInput,
						) (participant.SendQRCodesOutput, error) {
							capturedIsAdmin = isAdmin
							return participant.SendQRCodesOutput{SentCount: 2, Total: 2}, nil
						})

					r := newQRCodeHandlerRouter(mockUC, adminID, string(entity.RoleAdmin), log)

					reqBody := `{"send_to_all": true}`
					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(capturedIsAdmin).To(BeTrue())
				})
			})
		})

		When("checking the EventID passed to the usecase", func() {
			Context("when a specific event ID is in the URL path", func() {
				It("should pass the correct event ID to the usecase input", func() {
					specificEventID := uuid.New()
					var capturedInput participant.SendQRCodesInput

					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(
							_ context.Context, _ uuid.UUID, _ bool, input participant.SendQRCodesInput,
						) (participant.SendQRCodesOutput, error) {
							capturedInput = input
							return participant.SendQRCodesOutput{SentCount: 1, Total: 1}, nil
						})

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					reqBody := `{"send_to_all": true}`
					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+specificEventID.String()+"/qrcodes/send",
						strings.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(capturedInput.EventID).To(Equal(specificEventID))
				})
			})
		})

		When("participant_ids is provided with multiple UUIDs", func() {
			Context("when the list contains three participant IDs", func() {
				It("should correctly convert all three OpenAPI UUIDs to domain UUIDs", func() {
					pid1, pid2, pid3 := uuid.New(), uuid.New(), uuid.New()
					var capturedInput participant.SendQRCodesInput

					mockUC := participantMocks.NewMockUsecase(ctrl)
					mockUC.EXPECT().SendQRCodes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(
							_ context.Context, _ uuid.UUID, _ bool, input participant.SendQRCodesInput,
						) (participant.SendQRCodesOutput, error) {
							capturedInput = input
							return participant.SendQRCodesOutput{SentCount: 3, Total: 3}, nil
						})

					r := newQRCodeHandlerRouter(mockUC, userID, "organizer", log)

					pids := []openapi_types.UUID{
						openapi_types.UUID(pid1),
						openapi_types.UUID(pid2),
						openapi_types.UUID(pid3),
					}
					reqBody, err := json.Marshal(generated.SendQRCodesRequest{
						ParticipantIds: &pids,
					})
					Expect(err).NotTo(HaveOccurred())

					req := httptest.NewRequest(
						http.MethodPost,
						"/events/"+eventID.String()+"/qrcodes/send",
						strings.NewReader(string(reqBody)),
					)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(capturedInput.ParticipantIDs).To(HaveLen(3))
					Expect(capturedInput.ParticipantIDs).To(ConsistOf(pid1, pid2, pid3))
				})
			})
		})
	})
})

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool {
	return &b
}
