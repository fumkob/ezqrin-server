package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/handler"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/internal/usecase/event"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockEventUsecase is a hand-rolled mock for event.Usecase used in handler unit tests.
type mockEventUsecase struct {
	createFunc   func(ctx context.Context, input event.CreateEventInput) (*entity.Event, error)
	getByIDFunc  func(ctx context.Context, id uuid.UUID) (*entity.Event, error)
	listFunc     func(ctx context.Context, input event.ListEventsInput) (event.ListEventsOutput, error)
	updateFunc   func(ctx context.Context, id uuid.UUID, organizerID uuid.UUID, isAdmin bool, input event.UpdateEventInput) (*entity.Event, error)
	deleteFunc   func(ctx context.Context, id uuid.UUID, organizerID uuid.UUID, isAdmin bool) error
	getStatsFunc func(ctx context.Context, id uuid.UUID, organizerID uuid.UUID, isAdmin bool) (event.EventStatsOutput, error)
}

func (m *mockEventUsecase) Create(ctx context.Context, input event.CreateEventInput) (*entity.Event, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, input)
	}
	return nil, nil
}

func (m *mockEventUsecase) GetByID(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockEventUsecase) List(ctx context.Context, input event.ListEventsInput) (event.ListEventsOutput, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, input)
	}
	return event.ListEventsOutput{}, nil
}

func (m *mockEventUsecase) Update(ctx context.Context, id uuid.UUID, organizerID uuid.UUID, isAdmin bool, input event.UpdateEventInput) (*entity.Event, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, organizerID, isAdmin, input)
	}
	return nil, nil
}

func (m *mockEventUsecase) Delete(ctx context.Context, id uuid.UUID, organizerID uuid.UUID, isAdmin bool) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, organizerID, isAdmin)
	}
	return nil
}

func (m *mockEventUsecase) GetStats(ctx context.Context, id uuid.UUID, organizerID uuid.UUID, isAdmin bool) (event.EventStatsOutput, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx, id, organizerID, isAdmin)
	}
	return event.EventStatsOutput{}, nil
}

// newTestEntityEvent creates an entity.Event with specified participant and checked-in counts.
func newTestEntityEvent(organizerID uuid.UUID, participantCount, checkedInCount int64) *entity.Event {
	now := time.Now().UTC()
	return &entity.Event{
		ID:               uuid.New(),
		OrganizerID:      organizerID,
		Name:             "Test Event",
		StartDate:        now.Add(24 * time.Hour),
		Status:           entity.StatusPublished,
		Timezone:         "UTC",
		CreatedAt:        now,
		UpdatedAt:        now,
		ParticipantCount: participantCount,
		CheckedInCount:   checkedInCount,
	}
}

// newEventHandlerRouter creates a Gin router with EventHandler routes, injecting auth context.
func newEventHandlerRouter(uc event.Usecase, userID uuid.UUID, role string, log *logger.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Set(middleware.ContextKeyUserRole, role)
		c.Next()
	})

	h := handler.NewEventHandler(uc, log)

	r.GET("/events", func(c *gin.Context) {
		h.GetEvents(c, generated.GetEventsParams{})
	})
	r.GET("/events/:id", func(c *gin.Context) {
		id, _ := uuid.Parse(c.Param("id"))
		h.GetEventsId(c, id)
	})
	r.PUT("/events/:id", func(c *gin.Context) {
		id, _ := uuid.Parse(c.Param("id"))
		h.PutEventsId(c, id)
	})

	return r
}

var _ = Describe("EventHandler", func() {
	var (
		log         *logger.Logger
		organizerID uuid.UUID
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		var err error
		log, err = logger.New(logger.Config{
			Level:       "warn",
			Format:      "console",
			Environment: "test",
		})
		Expect(err).NotTo(HaveOccurred())
		organizerID = uuid.New()
	})

	// GetEvents returns: {"data": [...events...], "meta": {...}}
	Describe("GetEvents", func() {
		When("listing events", func() {
			Context("with events that have non-zero participant_count and checked_in_count", func() {
				It("should include participant_count and checked_in_count in each event in the response", func() {
					evt := newTestEntityEvent(organizerID, 42, 10)
					mockUC := &mockEventUsecase{
						listFunc: func(ctx context.Context, input event.ListEventsInput) (event.ListEventsOutput, error) {
							return event.ListEventsOutput{
								Events:     []*entity.Event{evt},
								TotalCount: 1,
							}, nil
						},
					}

					r := newEventHandlerRouter(mockUC, organizerID, "organizer", log)

					req := httptest.NewRequest(http.MethodGet, "/events", nil)
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())

					// Response format: {"data": [...], "meta": {...}}
					events, ok := body["data"].([]interface{})
					Expect(ok).To(BeTrue(), "top-level 'data' should be an array of events")
					Expect(events).To(HaveLen(1))

					firstEvent, ok := events[0].(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(firstEvent).To(HaveKey("participant_count"))
					Expect(firstEvent).To(HaveKey("checked_in_count"))
					Expect(firstEvent["participant_count"]).To(BeEquivalentTo(42))
					Expect(firstEvent["checked_in_count"]).To(BeEquivalentTo(10))
				})
			})

			Context("with events that have zero counts", func() {
				It("should include participant_count=0 and checked_in_count=0 in the response", func() {
					evt := newTestEntityEvent(organizerID, 0, 0)
					mockUC := &mockEventUsecase{
						listFunc: func(ctx context.Context, input event.ListEventsInput) (event.ListEventsOutput, error) {
							return event.ListEventsOutput{
								Events:     []*entity.Event{evt},
								TotalCount: 1,
							}, nil
						},
					}

					r := newEventHandlerRouter(mockUC, organizerID, "organizer", log)

					req := httptest.NewRequest(http.MethodGet, "/events", nil)
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())

					events := body["data"].([]interface{})
					Expect(events).To(HaveLen(1))

					firstEvent := events[0].(map[string]interface{})
					Expect(firstEvent).To(HaveKey("participant_count"))
					Expect(firstEvent).To(HaveKey("checked_in_count"))
					Expect(firstEvent["participant_count"]).To(BeEquivalentTo(0))
					Expect(firstEvent["checked_in_count"]).To(BeEquivalentTo(0))
				})
			})

			Context("with multiple events each having different counts", func() {
				It("should map counts correctly per event", func() {
					evt1 := newTestEntityEvent(organizerID, 5, 3)
					evt2 := newTestEntityEvent(organizerID, 20, 18)
					mockUC := &mockEventUsecase{
						listFunc: func(ctx context.Context, input event.ListEventsInput) (event.ListEventsOutput, error) {
							return event.ListEventsOutput{
								Events:     []*entity.Event{evt1, evt2},
								TotalCount: 2,
							}, nil
						},
					}

					r := newEventHandlerRouter(mockUC, organizerID, "organizer", log)

					req := httptest.NewRequest(http.MethodGet, "/events", nil)
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())

					events := body["data"].([]interface{})
					Expect(events).To(HaveLen(2))

					first := events[0].(map[string]interface{})
					Expect(first["participant_count"]).To(BeEquivalentTo(5))
					Expect(first["checked_in_count"]).To(BeEquivalentTo(3))

					second := events[1].(map[string]interface{})
					Expect(second["participant_count"]).To(BeEquivalentTo(20))
					Expect(second["checked_in_count"]).To(BeEquivalentTo(18))
				})
			})
		})
	})

	// GetEventsId returns the event object directly (no wrapper).
	Describe("GetEventsId", func() {
		When("getting a single event as its owner", func() {
			Context("when the event has participant_count and checked_in_count", func() {
				It("should include participant_count and checked_in_count in the response", func() {
					evt := newTestEntityEvent(organizerID, 15, 7)

					mockUC := &mockEventUsecase{
						getByIDFunc: func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
							return evt, nil
						},
					}

					r := newEventHandlerRouter(mockUC, organizerID, "organizer", log)

					req := httptest.NewRequest(http.MethodGet, "/events/"+evt.ID.String(), nil)
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					// Response is the event object directly.
					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
					Expect(body).To(HaveKey("participant_count"))
					Expect(body).To(HaveKey("checked_in_count"))
					Expect(body["participant_count"]).To(BeEquivalentTo(15))
					Expect(body["checked_in_count"]).To(BeEquivalentTo(7))
				})
			})
		})

		When("getting a single event as admin", func() {
			Context("when the event belongs to a different organizer", func() {
				It("should include participant_count and checked_in_count in the response", func() {
					adminID := uuid.New()
					evt := newTestEntityEvent(organizerID, 100, 55)

					mockUC := &mockEventUsecase{
						getByIDFunc: func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
							return evt, nil
						},
					}

					r := newEventHandlerRouter(mockUC, adminID, "admin", log)

					req := httptest.NewRequest(http.MethodGet, "/events/"+evt.ID.String(), nil)
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
					Expect(body).To(HaveKey("participant_count"))
					Expect(body).To(HaveKey("checked_in_count"))
					Expect(body["participant_count"]).To(BeEquivalentTo(100))
					Expect(body["checked_in_count"]).To(BeEquivalentTo(55))
				})
			})
		})
	})

	// PutEventsId returns the event object directly (no wrapper).
	Describe("PutEventsId", func() {
		When("updating an event as owner", func() {
			Context("when the updated event has participant_count and checked_in_count", func() {
				It("should include participant_count and checked_in_count in the response", func() {
					evt := newTestEntityEvent(organizerID, 30, 12)

					mockUC := &mockEventUsecase{
						updateFunc: func(ctx context.Context, id uuid.UUID, oID uuid.UUID, isAdmin bool, input event.UpdateEventInput) (*entity.Event, error) {
							return evt, nil
						},
					}

					r := newEventHandlerRouter(mockUC, organizerID, "organizer", log)

					reqBody := `{"name":"Updated Event","start_date":"2030-01-01T00:00:00Z","status":"published"}`
					req := httptest.NewRequest(http.MethodPut, "/events/"+evt.ID.String(), strings.NewReader(reqBody))
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
					Expect(body).To(HaveKey("participant_count"))
					Expect(body).To(HaveKey("checked_in_count"))
					Expect(body["participant_count"]).To(BeEquivalentTo(30))
					Expect(body["checked_in_count"]).To(BeEquivalentTo(12))
				})
			})
		})

		When("updating an event as admin", func() {
			Context("when the event belongs to a different organizer", func() {
				It("should include participant_count and checked_in_count in the response", func() {
					adminID := uuid.New()
					evt := newTestEntityEvent(organizerID, 8, 4)

					mockUC := &mockEventUsecase{
						updateFunc: func(ctx context.Context, id uuid.UUID, oID uuid.UUID, isAdmin bool, input event.UpdateEventInput) (*entity.Event, error) {
							return evt, nil
						},
					}

					r := newEventHandlerRouter(mockUC, adminID, "admin", log)

					reqBody := `{"name":"Admin Updated Event","start_date":"2030-06-01T00:00:00Z","status":"published"}`
					req := httptest.NewRequest(http.MethodPut, "/events/"+evt.ID.String(), strings.NewReader(reqBody))
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var body map[string]interface{}
					Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
					Expect(body).To(HaveKey("participant_count"))
					Expect(body).To(HaveKey("checked_in_count"))
					Expect(body["participant_count"]).To(BeEquivalentTo(8))
					Expect(body["checked_in_count"]).To(BeEquivalentTo(4))
				})
			})
		})
	})
})
