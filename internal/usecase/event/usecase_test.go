package event_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/internal/usecase/event"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventUsecase(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventUsecase Suite")
}

type eventListFunc func(
	ctx context.Context,
	filter repository.EventListFilter,
	offset, limit int,
) ([]*entity.Event, int64, error)

// SimpleEventRepositoryMock is a mock implementation of EventRepository for testing
type SimpleEventRepositoryMock struct {
	createFunc   func(ctx context.Context, event *entity.Event) error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (*entity.Event, error)
	listFunc     eventListFunc
	updateFunc   func(ctx context.Context, event *entity.Event) error
	deleteFunc   func(ctx context.Context, id uuid.UUID) error
	getStatsFunc func(ctx context.Context, id uuid.UUID) (*repository.EventStats, error)
}

func (m *SimpleEventRepositoryMock) Create(ctx context.Context, e *entity.Event) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, e)
	}
	return nil
}

func (m *SimpleEventRepositoryMock) FindByID(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *SimpleEventRepositoryMock) List(
	ctx context.Context,
	filter repository.EventListFilter,
	offset, limit int,
) ([]*entity.Event, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter, offset, limit)
	}
	return nil, 0, nil
}

func (m *SimpleEventRepositoryMock) Update(ctx context.Context, e *entity.Event) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, e)
	}
	return nil
}

func (m *SimpleEventRepositoryMock) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *SimpleEventRepositoryMock) GetStats(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx, id)
	}
	return nil, nil
}

func (m *SimpleEventRepositoryMock) HealthCheck(ctx context.Context) error {
	return nil
}

// Helper functions for pointer creation
func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func statusPtr(s entity.EventStatus) *entity.EventStatus {
	return &s
}

// Helper functions for test data creation
func newValidEvent(organizerID uuid.UUID) *entity.Event {
	return &entity.Event{
		ID:          uuid.New(),
		OrganizerID: organizerID,
		Name:        "Test Event",
		Description: "Test Description",
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     timePtr(time.Now().Add(48 * time.Hour)),
		Location:    "Test Location",
		Timezone:    "UTC",
		Status:      entity.StatusDraft,
	}
}

func newValidCreateInput(organizerID uuid.UUID) event.CreateEventInput {
	return event.CreateEventInput{
		OrganizerID: organizerID,
		Name:        "Test Event",
		Description: "Test Description",
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     timePtr(time.Now().Add(48 * time.Hour)),
		Location:    "Test Location",
		Timezone:    "UTC",
		Status:      entity.StatusDraft,
	}
}

var _ = Describe("EventUsecase", func() {
	var (
		mockRepo  *SimpleEventRepositoryMock
		usecase   event.Usecase
		ctx       context.Context
		eventID   uuid.UUID
		userID    uuid.UUID
		adminID   uuid.UUID
		testEvent *entity.Event
	)

	BeforeEach(func() {
		mockRepo = &SimpleEventRepositoryMock{}
		usecase = event.NewUsecase(mockRepo)
		ctx = context.Background()

		eventID = uuid.New()
		userID = uuid.New()
		adminID = uuid.New()

		testEvent = newValidEvent(userID)
		testEvent.ID = eventID
	})

	Describe("Create", func() {
		When("creating with valid input", func() {
			Context("with all required fields", func() {
				It("should create event successfully", func() {
					input := newValidCreateInput(userID)
					mockRepo.createFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Create(ctx, input)

					Expect(err).To(BeNil())
					Expect(result).NotTo(BeNil())
					Expect(result.ID).NotTo(Equal(uuid.Nil))
					Expect(result.Name).To(Equal(input.Name))
					Expect(result.OrganizerID).To(Equal(userID))
					Expect(result.Status).To(Equal(entity.StatusDraft))
				})
			})

			Context("with optional EndDate", func() {
				It("should create event with EndDate", func() {
					input := newValidCreateInput(userID)
					endDate := time.Now().Add(48 * time.Hour)
					input.EndDate = &endDate

					mockRepo.createFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Create(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.EndDate).NotTo(BeNil())
					Expect(result.EndDate.Unix()).To(Equal(endDate.Unix()))
				})
			})

			Context("with draft status", func() {
				It("should create event with draft status", func() {
					input := newValidCreateInput(userID)
					input.Status = entity.StatusDraft

					mockRepo.createFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Create(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.Status).To(Equal(entity.StatusDraft))
				})
			})

			Context("with published status", func() {
				It("should create event with published status", func() {
					input := newValidCreateInput(userID)
					input.Status = entity.StatusPublished

					mockRepo.createFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Create(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.Status).To(Equal(entity.StatusPublished))
				})
			})
		})

		When("validation fails", func() {
			Context("with empty name", func() {
				It("should return validation error", func() {
					input := newValidCreateInput(userID)
					input.Name = ""

					result, err := usecase.Create(ctx, input)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
					Expect(result).To(BeNil())
				})
			})

			Context("with name too long (256 characters)", func() {
				It("should return validation error", func() {
					input := newValidCreateInput(userID)
					input.Name = string(make([]byte, 256))

					_, err := usecase.Create(ctx, input)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
				})
			})

			Context("with description too long (5001 characters)", func() {
				It("should return validation error", func() {
					input := newValidCreateInput(userID)
					input.Description = string(make([]byte, 5001))

					_, err := usecase.Create(ctx, input)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
				})
			})

			Context("with zero StartDate", func() {
				It("should return validation error", func() {
					input := newValidCreateInput(userID)
					input.StartDate = time.Time{}

					_, err := usecase.Create(ctx, input)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
				})
			})

			Context("with EndDate before StartDate", func() {
				It("should return validation error", func() {
					input := newValidCreateInput(userID)
					input.StartDate = time.Now().Add(48 * time.Hour)
					input.EndDate = timePtr(time.Now().Add(24 * time.Hour))

					_, err := usecase.Create(ctx, input)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
				})
			})

			Context("with location too long (501 characters)", func() {
				It("should return validation error", func() {
					input := newValidCreateInput(userID)
					input.Location = string(make([]byte, 501))

					_, err := usecase.Create(ctx, input)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
				})
			})
		})

		When("repository fails", func() {
			Context("with database error", func() {
				It("should return wrapped error", func() {
					input := newValidCreateInput(userID)
					dbErr := errors.New("database connection error")

					mockRepo.createFunc = func(ctx context.Context, e *entity.Event) error {
						return dbErr
					}

					result, err := usecase.Create(ctx, input)

					Expect(err).NotTo(BeNil())
					Expect(errors.Is(err, dbErr)).To(BeTrue())
					Expect(result).To(BeNil())
				})
			})
		})

		When("context is cancelled", func() {
			It("should propagate context error", func() {
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()

				input := newValidCreateInput(userID)

				mockRepo.createFunc = func(ctx context.Context, e *entity.Event) error {
					return cancelledCtx.Err()
				}

				result, err := usecase.Create(cancelledCtx, input)

				Expect(err).NotTo(BeNil())
				Expect(errors.Is(err, context.Canceled)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("GetByID", func() {
		When("event exists", func() {
			Context("with draft status", func() {
				It("should return event", func() {
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					result, err := usecase.GetByID(ctx, eventID)

					Expect(err).To(BeNil())
					Expect(result).NotTo(BeNil())
					Expect(result.ID).To(Equal(eventID))
					Expect(result.Status).To(Equal(entity.StatusDraft))
				})
			})

			Context("with different status values", func() {
				It("should return event regardless of status", func() {
					for _, status := range []entity.EventStatus{
						entity.StatusDraft,
						entity.StatusPublished,
						entity.StatusOngoing,
						entity.StatusCompleted,
						entity.StatusCancelled,
					} {
						testEvent.Status = status
						mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
							return testEvent, nil
						}

						result, err := usecase.GetByID(ctx, eventID)

						Expect(err).To(BeNil())
						Expect(result.Status).To(Equal(status))
					}
				})
			})
		})

		When("event does not exist", func() {
			It("should return not found error", func() {
				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return nil, apperrors.NotFound("event not found")
				}

				result, err := usecase.GetByID(ctx, eventID)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		When("repository fails", func() {
			Context("with database error", func() {
				It("should return error", func() {
					dbErr := errors.New("database connection error")
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return nil, dbErr
					}

					result, err := usecase.GetByID(ctx, eventID)

					Expect(err).NotTo(BeNil())
					Expect(errors.Is(err, dbErr)).To(BeTrue())
					Expect(result).To(BeNil())
				})
			})
		})

		When("context is cancelled", func() {
			It("should propagate context error", func() {
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()

				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return nil, cancelledCtx.Err()
				}

				_, err := usecase.GetByID(cancelledCtx, eventID)

				Expect(err).NotTo(BeNil())
				Expect(errors.Is(err, context.Canceled)).To(BeTrue())
			})
		})

		When("verifying return value", func() {
			It("should return exact event from repository", func() {
				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return testEvent, nil
				}

				result, err := usecase.GetByID(ctx, eventID)

				Expect(err).To(BeNil())
				Expect(result).To(Equal(testEvent))
			})
		})
	})

	Describe("List", func() {
		When("listing with no filters", func() {
			Context("listing all events", func() {
				It("should return all events", func() {
					events := []*entity.Event{newValidEvent(userID), newValidEvent(userID)}
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.Events).To(HaveLen(2))
					Expect(result.TotalCount).To(Equal(int64(2)))
				})
			})

			Context("with pagination", func() {
				It("should calculate correct offset", func() {
					events := []*entity.Event{newValidEvent(userID)}
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						Expect(offset).To(Equal(0))
						Expect(limit).To(Equal(10))
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						Page:    1,
						PerPage: 10,
					}

					_, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
				})
			})

			Context("with empty results", func() {
				It("should return empty list", func() {
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						return []*entity.Event{}, 0, nil
					}

					input := event.ListEventsInput{
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.Events).To(HaveLen(0))
					Expect(result.TotalCount).To(Equal(int64(0)))
				})
			})
		})

		When("filtering by OrganizerID", func() {
			Context("with organizer filter", func() {
				It("should pass organizer filter to repository", func() {
					events := []*entity.Event{newValidEvent(userID)}
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						Expect(filter.OrganizerID).NotTo(BeNil())
						Expect(*filter.OrganizerID).To(Equal(userID))
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						OrganizerID: &userID,
						Page:        1,
						PerPage:     10,
					}

					result, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.Events).To(HaveLen(1))
				})
			})
		})

		When("filtering by Status", func() {
			Context("with draft status", func() {
				It("should pass status filter to repository", func() {
					draftEvent := newValidEvent(userID)
					draftEvent.Status = entity.StatusDraft
					events := []*entity.Event{draftEvent}
					status := entity.StatusDraft

					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						Expect(filter.Status).NotTo(BeNil())
						Expect(*filter.Status).To(Equal(entity.StatusDraft))
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						Status:  &status,
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.Events).To(HaveLen(1))
				})
			})

			Context("with published status", func() {
				It("should filter by published status", func() {
					publishedEvent := newValidEvent(userID)
					publishedEvent.Status = entity.StatusPublished
					events := []*entity.Event{publishedEvent}
					status := entity.StatusPublished

					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						Status:  &status,
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.Events).To(HaveLen(1))
				})
			})
		})

		When("filtering by Search", func() {
			Context("with search term", func() {
				It("should pass search filter to repository", func() {
					events := []*entity.Event{newValidEvent(userID)}
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						Expect(filter.Search).To(Equal("test"))
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						Search:  "test",
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.Events).To(HaveLen(1))
				})
			})
		})

		When("using pagination", func() {
			Context("page 1 with limit 10", func() {
				It("should use offset 0", func() {
					events := []*entity.Event{newValidEvent(userID)}
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						Expect(offset).To(Equal(0))
						Expect(limit).To(Equal(10))
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						Page:    1,
						PerPage: 10,
					}

					_, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
				})
			})

			Context("page 2 with limit 10", func() {
				It("should use offset 10", func() {
					events := []*entity.Event{newValidEvent(userID)}
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						Expect(offset).To(Equal(10))
						Expect(limit).To(Equal(10))
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						Page:    2,
						PerPage: 10,
					}

					_, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
				})
			})

			Context("page 3 with limit 5", func() {
				It("should use offset 10", func() {
					events := []*entity.Event{newValidEvent(userID)}
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						Expect(offset).To(Equal(10))
						Expect(limit).To(Equal(5))
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						Page:    3,
						PerPage: 5,
					}

					_, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
				})
			})
		})

		When("repository returns results", func() {
			Context("with matching count", func() {
				It("should preserve totalCount", func() {
					events := []*entity.Event{newValidEvent(userID), newValidEvent(userID)}
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						return events, 100, nil
					}

					input := event.ListEventsInput{
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.TotalCount).To(Equal(int64(100)))
				})
			})

			Context("with events array", func() {
				It("should preserve events list", func() {
					event1 := newValidEvent(userID)
					event2 := newValidEvent(userID)
					events := []*entity.Event{event1, event2}
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						return events, int64(len(events)), nil
					}

					input := event.ListEventsInput{
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, input)

					Expect(err).To(BeNil())
					Expect(result.Events).To(Equal(events))
				})
			})
		})

		When("repository fails", func() {
			Context("with database error", func() {
				It("should return wrapped error", func() {
					dbErr := errors.New("database connection error")
					mockRepo.listFunc = func(
						ctx context.Context,
						filter repository.EventListFilter,
						offset, limit int,
					) ([]*entity.Event, int64, error) {
						return nil, 0, dbErr
					}

					input := event.ListEventsInput{
						Page:    1,
						PerPage: 10,
					}

					_, err := usecase.List(ctx, input)

					Expect(err).NotTo(BeNil())
					Expect(errors.Is(err, dbErr)).To(BeTrue())
				})
			})
		})

		When("context is cancelled", func() {
			It("should propagate context error", func() {
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()

				mockRepo.listFunc = func(
					ctx context.Context,
					filter repository.EventListFilter,
					offset, limit int,
				) ([]*entity.Event, int64, error) {
					return nil, 0, cancelledCtx.Err()
				}

				input := event.ListEventsInput{
					Page:    1,
					PerPage: 10,
				}

				_, err := usecase.List(cancelledCtx, input)

				Expect(err).NotTo(BeNil())
				Expect(errors.Is(err, context.Canceled)).To(BeTrue())
			})
		})
	})

	Describe("GetStats", func() {
		When("getting stats as owner", func() {
			Context("with checked in participants", func() {
				It("should calculate stats correctly", func() {
					stats := &repository.EventStats{
						TotalParticipants: 10,
						CheckedInCount:    8,
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.getStatsFunc = func(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
						return stats, nil
					}

					result, err := usecase.GetStats(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
					Expect(result.EventID).To(Equal(eventID))
					Expect(result.TotalParticipants).To(Equal(int64(10)))
					Expect(result.CheckedInParticipants).To(Equal(int64(8)))
					Expect(result.CheckinRate).To(BeNumerically("~", 0.8, 0.0001))
				})
			})

			Context("with zero participants", func() {
				It("should avoid division by zero", func() {
					stats := &repository.EventStats{
						TotalParticipants: 0,
						CheckedInCount:    0,
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.getStatsFunc = func(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
						return stats, nil
					}

					result, err := usecase.GetStats(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
					Expect(result.CheckinRate).To(Equal(0.0))
				})
			})

			Context("with no checked in participants", func() {
				It("should calculate zero rate", func() {
					stats := &repository.EventStats{
						TotalParticipants: 10,
						CheckedInCount:    0,
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.getStatsFunc = func(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
						return stats, nil
					}

					result, err := usecase.GetStats(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
					Expect(result.CheckinRate).To(Equal(0.0))
				})
			})

			Context("with all participants checked in", func() {
				It("should calculate rate of 1.0", func() {
					stats := &repository.EventStats{
						TotalParticipants: 5,
						CheckedInCount:    5,
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.getStatsFunc = func(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
						return stats, nil
					}

					result, err := usecase.GetStats(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
					Expect(result.CheckinRate).To(Equal(1.0))
				})
			})

			Context("with partial check-in (rate 0.5)", func() {
				It("should calculate correct rate", func() {
					stats := &repository.EventStats{
						TotalParticipants: 10,
						CheckedInCount:    5,
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.getStatsFunc = func(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
						return stats, nil
					}

					result, err := usecase.GetStats(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
					Expect(result.CheckinRate).To(Equal(0.5))
				})
			})
		})

		When("authorization fails", func() {
			Context("as non-owner with organizer role", func() {
				It("should return forbidden error", func() {
					otherUserID := uuid.New()
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					result, err := usecase.GetStats(ctx, eventID, otherUserID, false)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsForbidden(err)).To(BeTrue())
					Expect(err.Error()).To(ContainSubstring("you do not have permission to view stats for this event"))
					Expect(result.EventID).To(Equal(uuid.Nil))
				})
			})
		})

		When("getting stats as admin", func() {
			Context("with other user's event", func() {
				It("should retrieve stats successfully", func() {
					stats := &repository.EventStats{
						TotalParticipants: 20,
						CheckedInCount:    15,
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.getStatsFunc = func(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
						return stats, nil
					}

					result, err := usecase.GetStats(ctx, eventID, adminID, true)

					Expect(err).To(BeNil())
					Expect(result.TotalParticipants).To(Equal(int64(20)))
					Expect(result.CheckedInParticipants).To(Equal(int64(15)))
				})
			})
		})

		When("calculating checkin rate", func() {
			Context("with non-zero participants", func() {
				It("should calculate correct percentage", func() {
					stats := &repository.EventStats{
						TotalParticipants: 4,
						CheckedInCount:    1,
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.getStatsFunc = func(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
						return stats, nil
					}

					result, err := usecase.GetStats(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
					Expect(result.CheckinRate).To(Equal(0.25))
				})
			})
		})

		When("event does not exist", func() {
			It("should return not found error", func() {
				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return nil, apperrors.NotFound("event not found")
				}

				result, err := usecase.GetStats(ctx, eventID, userID, false)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(result.EventID).To(Equal(uuid.Nil))
			})
		})

		When("repository GetStats fails", func() {
			It("should return wrapped error", func() {
				dbErr := errors.New("database connection error")
				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return testEvent, nil
				}

				mockRepo.getStatsFunc = func(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
					return nil, dbErr
				}

				result, err := usecase.GetStats(ctx, eventID, userID, false)

				Expect(err).NotTo(BeNil())
				Expect(errors.Is(err, dbErr)).To(BeTrue())
				Expect(result.EventID).To(Equal(uuid.Nil))
			})
		})

		When("context is cancelled", func() {
			It("should propagate context error", func() {
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()

				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return nil, cancelledCtx.Err()
				}

				result, err := usecase.GetStats(cancelledCtx, eventID, userID, false)

				Expect(err).NotTo(BeNil())
				Expect(errors.Is(err, context.Canceled)).To(BeTrue())
				Expect(result.EventID).To(Equal(uuid.Nil))
			})
		})
	})

	Describe("Update", func() {
		When("updating as owner", func() {
			Context("updating name only", func() {
				It("should update only name", func() {
					updateInput := event.UpdateEventInput{
						Name: strPtr("Updated Name"),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						Expect(e.Name).To(Equal("Updated Name"))
						Expect(e.Description).To(Equal(testEvent.Description))
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Name).To(Equal("Updated Name"))
				})
			})

			Context("updating description only", func() {
				It("should update only description", func() {
					updateInput := event.UpdateEventInput{
						Description: strPtr("Updated Description"),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						Expect(e.Description).To(Equal("Updated Description"))
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Description).To(Equal("Updated Description"))
				})
			})

			Context("updating dates", func() {
				It("should update start and end dates", func() {
					newStart := time.Now().Add(72 * time.Hour)
					newEnd := time.Now().Add(96 * time.Hour)
					updateInput := event.UpdateEventInput{
						StartDate: &newStart,
						EndDate:   &newEnd,
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.StartDate.Unix()).To(Equal(newStart.Unix()))
				})
			})

			Context("updating location", func() {
				It("should update location", func() {
					updateInput := event.UpdateEventInput{
						Location: strPtr("New Location"),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Location).To(Equal("New Location"))
				})
			})

			Context("updating timezone", func() {
				It("should update timezone", func() {
					updateInput := event.UpdateEventInput{
						Timezone: strPtr("America/New_York"),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Timezone).To(Equal("America/New_York"))
				})
			})

			Context("updating multiple fields", func() {
				It("should update all provided fields", func() {
					updateInput := event.UpdateEventInput{
						Name:        strPtr("Updated Event"),
						Description: strPtr("New Description"),
						Location:    strPtr("New Location"),
						Timezone:    strPtr("UTC"),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Name).To(Equal("Updated Event"))
					Expect(result.Description).To(Equal("New Description"))
					Expect(result.Location).To(Equal("New Location"))
				})
			})

			Context("with empty update (no changes)", func() {
				It("should succeed with no modifications", func() {
					updateInput := event.UpdateEventInput{}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result).NotTo(BeNil())
				})
			})
		})

		When("updating status as owner", func() {
			Context("draft to published", func() {
				It("should transition successfully", func() {
					testEvent.Status = entity.StatusDraft
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusPublished),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Status).To(Equal(entity.StatusPublished))
				})
			})

			Context("draft to cancelled", func() {
				It("should transition successfully", func() {
					testEvent.Status = entity.StatusDraft
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusCancelled),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Status).To(Equal(entity.StatusCancelled))
				})
			})

			Context("published to ongoing", func() {
				It("should transition successfully", func() {
					testEvent.Status = entity.StatusPublished
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusOngoing),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Status).To(Equal(entity.StatusOngoing))
				})
			})

			Context("published to cancelled", func() {
				It("should transition successfully", func() {
					testEvent.Status = entity.StatusPublished
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusCancelled),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Status).To(Equal(entity.StatusCancelled))
				})
			})

			Context("ongoing to completed", func() {
				It("should transition successfully", func() {
					testEvent.Status = entity.StatusOngoing
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusCompleted),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Status).To(Equal(entity.StatusCompleted))
				})
			})

			Context("ongoing to cancelled", func() {
				It("should transition successfully", func() {
					testEvent.Status = entity.StatusOngoing
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusCancelled),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Status).To(Equal(entity.StatusCancelled))
				})
			})

			Context("draft to ongoing (invalid)", func() {
				It("should return bad request error", func() {
					testEvent.Status = entity.StatusDraft
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusOngoing),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					_, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("invalid status transition"))
				})
			})

			Context("published to draft (invalid)", func() {
				It("should return bad request error", func() {
					testEvent.Status = entity.StatusPublished
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusDraft),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					_, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("invalid status transition"))
				})
			})

			Context("completed to any (invalid)", func() {
				It("should return bad request error", func() {
					testEvent.Status = entity.StatusCompleted
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusCancelled),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					_, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("invalid status transition"))
				})
			})

			Context("cancelled to any (invalid)", func() {
				It("should return bad request error", func() {
					testEvent.Status = entity.StatusCancelled
					updateInput := event.UpdateEventInput{
						Status: statusPtr(entity.StatusDraft),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					_, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("invalid status transition"))
				})
			})
		})

		When("authorization fails", func() {
			Context("non-owner trying to update", func() {
				It("should return forbidden error", func() {
					otherUserID := uuid.New()
					updateInput := event.UpdateEventInput{
						Name: strPtr("Hacked"),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					result, err := usecase.Update(ctx, eventID, otherUserID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsForbidden(err)).To(BeTrue())
					Expect(err.Error()).To(ContainSubstring("you do not have permission to update this event"))
					Expect(result).To(BeNil())
				})
			})
		})

		When("updating as admin", func() {
			Context("updating other user's event", func() {
				It("should succeed", func() {
					updateInput := event.UpdateEventInput{
						Name: strPtr("Admin Updated"),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, adminID, true, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Name).To(Equal("Admin Updated"))
				})
			})

			Context("admin can change all fields", func() {
				It("should allow all field changes", func() {
					updateInput := event.UpdateEventInput{
						Name:        strPtr("Admin Name"),
						Description: strPtr("Admin Desc"),
						Location:    strPtr("Admin Location"),
						Timezone:    strPtr("UTC"),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
						return nil
					}

					result, err := usecase.Update(ctx, eventID, adminID, true, updateInput)

					Expect(err).To(BeNil())
					Expect(result.Name).To(Equal("Admin Name"))
				})
			})
		})

		When("validation fails after update", func() {
			Context("with name too long", func() {
				It("should return validation error", func() {
					updateInput := event.UpdateEventInput{
						Name: strPtr(string(make([]byte, 256))),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
					Expect(result).To(BeNil())
				})
			})

			Context("with description too long", func() {
				It("should return validation error", func() {
					updateInput := event.UpdateEventInput{
						Description: strPtr(string(make([]byte, 5001))),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					_, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
				})
			})

			Context("with EndDate before StartDate", func() {
				It("should return validation error", func() {
					newStart := time.Now().Add(48 * time.Hour)
					newEnd := time.Now().Add(24 * time.Hour)
					updateInput := event.UpdateEventInput{
						StartDate: &newStart,
						EndDate:   &newEnd,
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					_, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
				})
			})

			Context("with location too long", func() {
				It("should return validation error", func() {
					updateInput := event.UpdateEventInput{
						Location: strPtr(string(make([]byte, 501))),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					_, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
				})
			})

			Context("with empty name", func() {
				It("should return validation error", func() {
					updateInput := event.UpdateEventInput{
						Name: strPtr(""),
					}

					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					_, err := usecase.Update(ctx, eventID, userID, false, updateInput)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsValidation(err)).To(BeTrue())
				})
			})
		})

		When("event does not exist", func() {
			It("should return not found error", func() {
				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return nil, apperrors.NotFound("event not found")
				}

				updateInput := event.UpdateEventInput{
					Name: strPtr("Updated"),
				}

				result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		When("repository update fails", func() {
			It("should return wrapped error", func() {
				dbErr := errors.New("database connection error")
				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return testEvent, nil
				}

				mockRepo.updateFunc = func(ctx context.Context, e *entity.Event) error {
					return dbErr
				}

				updateInput := event.UpdateEventInput{
					Name: strPtr("Updated"),
				}

				result, err := usecase.Update(ctx, eventID, userID, false, updateInput)

				Expect(err).NotTo(BeNil())
				Expect(errors.Is(err, dbErr)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		When("context is cancelled", func() {
			It("should propagate context error", func() {
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()

				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return nil, cancelledCtx.Err()
				}

				updateInput := event.UpdateEventInput{
					Name: strPtr("Updated"),
				}

				result, err := usecase.Update(cancelledCtx, eventID, userID, false, updateInput)

				Expect(err).NotTo(BeNil())
				Expect(errors.Is(err, context.Canceled)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("Delete", func() {
		When("deleting an event as owner", func() {
			Context("with draft status", func() {
				It("should delete successfully", func() {
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}
					mockRepo.deleteFunc = func(ctx context.Context, id uuid.UUID) error {
						return nil
					}

					err := usecase.Delete(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
				})
			})

			Context("with published status", func() {
				It("should delete successfully", func() {
					testEvent.Status = entity.StatusPublished
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}
					mockRepo.deleteFunc = func(ctx context.Context, id uuid.UUID) error {
						return nil
					}

					err := usecase.Delete(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
				})
			})

			Context("with completed status", func() {
				It("should delete successfully", func() {
					testEvent.Status = entity.StatusCompleted
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}
					mockRepo.deleteFunc = func(ctx context.Context, id uuid.UUID) error {
						return nil
					}

					err := usecase.Delete(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
				})
			})

			Context("with cancelled status", func() {
				It("should delete successfully", func() {
					testEvent.Status = entity.StatusCancelled
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}
					mockRepo.deleteFunc = func(ctx context.Context, id uuid.UUID) error {
						return nil
					}

					err := usecase.Delete(ctx, eventID, userID, false)

					Expect(err).To(BeNil())
				})
			})

			Context("with ongoing status", func() {
				It("should return conflict error", func() {
					testEvent.Status = entity.StatusOngoing
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					err := usecase.Delete(ctx, eventID, userID, false)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsConflict(err)).To(BeTrue())
					Expect(err.Error()).To(ContainSubstring("cannot delete event with status 'ongoing'"))
				})
			})
		})

		When("deleting an event as non-owner with organizer role", func() {
			Context("with draft status", func() {
				It("should return forbidden error", func() {
					otherUserID := uuid.New()
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					err := usecase.Delete(ctx, eventID, otherUserID, false)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsForbidden(err)).To(BeTrue())
					Expect(err.Error()).To(ContainSubstring("you do not have permission to delete this event"))
				})
			})
		})

		When("deleting an event as admin", func() {
			Context("with draft status", func() {
				It("should delete successfully even if not owner", func() {
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}
					mockRepo.deleteFunc = func(ctx context.Context, id uuid.UUID) error {
						return nil
					}

					err := usecase.Delete(ctx, eventID, adminID, true)

					Expect(err).To(BeNil())
				})
			})

			Context("with ongoing status", func() {
				It("should still return conflict error", func() {
					testEvent.Status = entity.StatusOngoing
					mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
						return testEvent, nil
					}

					err := usecase.Delete(ctx, eventID, adminID, true)

					Expect(err).NotTo(BeNil())
					Expect(apperrors.IsConflict(err)).To(BeTrue())
					Expect(err.Error()).To(ContainSubstring("cannot delete event with status 'ongoing'"))
				})
			})
		})

		When("event does not exist", func() {
			It("should return not found error", func() {
				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return nil, apperrors.NotFound("event not found")
				}

				err := usecase.Delete(ctx, eventID, userID, false)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
			})
		})

		When("repository delete fails", func() {
			It("should return wrapped error", func() {
				dbErr := errors.New("database connection error")
				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return testEvent, nil
				}
				mockRepo.deleteFunc = func(ctx context.Context, id uuid.UUID) error {
					return dbErr
				}

				err := usecase.Delete(ctx, eventID, userID, false)

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("failed to delete event"))
				Expect(errors.Is(err, dbErr)).To(BeTrue())
			})
		})

		When("context is cancelled", func() {
			It("should propagate context error", func() {
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()

				mockRepo.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
					return nil, cancelledCtx.Err()
				}

				err := usecase.Delete(cancelledCtx, eventID, userID, false)

				Expect(err).NotTo(BeNil())
				Expect(errors.Is(err, context.Canceled)).To(BeTrue())
			})
		})
	})
})
