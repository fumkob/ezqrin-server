package checkin_test

import (
	"context"
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/internal/usecase/checkin"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CheckIn UseCase", func() {
	var (
		ctx             context.Context
		usecase         checkin.Usecase
		mockCheckinRepo *mockCheckinRepository
		mockParticipant *mockParticipantRepository
		mockEventRepo   *mockEventRepository
		testEventID     uuid.UUID
		testUserID      uuid.UUID
		testOrganizerID uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		testEventID = uuid.New()
		testUserID = uuid.New()
		testOrganizerID = uuid.New()

		mockCheckinRepo = newMockCheckinRepository()
		mockParticipant = newMockParticipantRepository()
		mockEventRepo = newMockEventRepository()

		usecase = checkin.NewUsecase(mockCheckinRepo, mockParticipant, mockEventRepo)
	})

	Describe("CheckIn", func() {
		When("checking in with QR code", func() {
			Context("with valid QR code", func() {
				It("should successfully check in participant", func() {
					qrCode := "valid-qr-code-123"
					participant := &entity.Participant{
						ID:      uuid.New(),
						EventID: testEventID,
						Name:    "John Doe",
						Email:   "john@example.com",
						QRCode:  qrCode,
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID,
						Name:        "Test Event",
					}

					mockEventRepo.events[testEventID] = event
					mockParticipant.participants[qrCode] = participant
					mockCheckinRepo.existsMap[participant.ID] = false

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodQRCode,
						QRCode:      &qrCode,
						CheckedInBy: testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.ParticipantID).To(Equal(participant.ID))
					Expect(result.ParticipantName).To(Equal("John Doe"))
					Expect(result.Method).To(Equal(entity.CheckinMethodQRCode))
					Expect(mockCheckinRepo.created).To(HaveLen(1))
				})
			})

			Context("with invalid QR code", func() {
				It("should return not found error", func() {
					qrCode := "invalid-qr-code"
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID,
						Name:        "Test Event",
					}

					mockEventRepo.events[testEventID] = event

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodQRCode,
						QRCode:      &qrCode,
						CheckedInBy: testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeNotFound))
				})
			})

			Context("with missing QR code parameter", func() {
				It("should return bad request error", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID,
					}
					mockEventRepo.events[testEventID] = event

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodQRCode,
						QRCode:      nil, // Missing QR code
						CheckedInBy: testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
				})
			})
		})

		When("checking in manually", func() {
			Context("as event organizer", func() {
				It("should successfully check in participant", func() {
					participantID := uuid.New()
					participant := &entity.Participant{
						ID:      participantID,
						EventID: testEventID,
						Name:    "Jane Smith",
						Email:   "jane@example.com",
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID, // User is the organizer
						Name:        "Test Event",
					}

					mockEventRepo.events[testEventID] = event
					mockParticipant.participantsByID[participantID] = participant
					mockCheckinRepo.existsMap[participantID] = false

					input := checkin.CheckInInput{
						EventID:       testEventID,
						Method:        entity.CheckinMethodManual,
						ParticipantID: &participantID,
						CheckedInBy:   testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.ParticipantID).To(Equal(participantID))
					Expect(result.Method).To(Equal(entity.CheckinMethodManual))
					Expect(result.CheckedInBy).NotTo(BeNil())
					Expect(*result.CheckedInBy).To(Equal(testUserID))
				})
			})

			Context("as admin", func() {
				It("should successfully check in participant", func() {
					participantID := uuid.New()
					participant := &entity.Participant{
						ID:      participantID,
						EventID: testEventID,
						Name:    "Admin Checkin",
						Email:   "admin@example.com",
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID, // Different from current user
						Name:        "Test Event",
					}

					mockEventRepo.events[testEventID] = event
					mockParticipant.participantsByID[participantID] = participant
					mockCheckinRepo.existsMap[participantID] = false

					input := checkin.CheckInInput{
						EventID:       testEventID,
						Method:        entity.CheckinMethodManual,
						ParticipantID: &participantID,
						CheckedInBy:   testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, true, input) // isAdmin = true

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
				})
			})

			Context("without permission", func() {
				It("should return forbidden error", func() {
					participantID := uuid.New()
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID, // Different from current user
						Name:        "Test Event",
					}

					mockEventRepo.events[testEventID] = event

					input := checkin.CheckInInput{
						EventID:       testEventID,
						Method:        entity.CheckinMethodManual,
						ParticipantID: &participantID,
						CheckedInBy:   testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input) // Not admin, not organizer

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeForbidden))
				})
			})

			Context("with missing participant ID", func() {
				It("should return bad request error", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
					}
					mockEventRepo.events[testEventID] = event

					input := checkin.CheckInInput{
						EventID:       testEventID,
						Method:        entity.CheckinMethodManual,
						ParticipantID: nil, // Missing participant ID
						CheckedInBy:   testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
				})
			})
		})

		When("participant already checked in", func() {
			It("should return conflict error", func() {
				participantID := uuid.New()
				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Already Checked In",
					Email:   "already@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				mockEventRepo.events[testEventID] = event
				mockParticipant.participantsByID[participantID] = participant
				mockCheckinRepo.existsMap[participantID] = true // Already checked in

				input := checkin.CheckInInput{
					EventID:       testEventID,
					Method:        entity.CheckinMethodManual,
					ParticipantID: &participantID,
					CheckedInBy:   testUserID,
				}

				result, err := usecase.CheckIn(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeConflict))
			})
		})

		When("participant belongs to different event", func() {
			It("should return bad request error", func() {
				differentEventID := uuid.New()
				participantID := uuid.New()
				participant := &entity.Participant{
					ID:      participantID,
					EventID: differentEventID, // Different event
					Name:    "Wrong Event",
					Email:   "wrong@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				mockEventRepo.events[testEventID] = event
				mockParticipant.participantsByID[participantID] = participant
				mockCheckinRepo.existsMap[participantID] = false

				input := checkin.CheckInInput{
					EventID:       testEventID,
					Method:        entity.CheckinMethodManual,
					ParticipantID: &participantID,
					CheckedInBy:   testUserID,
				}

				result, err := usecase.CheckIn(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
			})
		})

		When("event does not exist", func() {
			It("should return not found error", func() {
				participantID := uuid.New()
				input := checkin.CheckInInput{
					EventID:       testEventID,
					Method:        entity.CheckinMethodManual,
					ParticipantID: &participantID,
					CheckedInBy:   testUserID,
				}

				result, err := usecase.CheckIn(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		When("invalid check-in method", func() {
			It("should return bad request error", func() {
				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
				}
				mockEventRepo.events[testEventID] = event

				input := checkin.CheckInInput{
					EventID:     testEventID,
					Method:      entity.CheckinMethod("invalid"),
					CheckedInBy: testUserID,
				}

				result, err := usecase.CheckIn(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
			})
		})
	})

	Describe("GetStatus", func() {
		When("participant has checked in", func() {
			It("should return checked-in status with details", func() {
				participantID := uuid.New()
				checkinID := uuid.New()

				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Checked In User",
					Email:   "checkedin@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				checkinRecord := &entity.Checkin{
					ID:            checkinID,
					EventID:       testEventID,
					ParticipantID: participantID,
					CheckedInAt:   time.Now(),
					Method:        entity.CheckinMethodQRCode,
				}

				mockEventRepo.events[testEventID] = event
				mockParticipant.participantsByID[participantID] = participant
				mockCheckinRepo.checkinsByParticipant[participantID] = checkinRecord

				result, err := usecase.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.IsCheckedIn).To(BeTrue())
				Expect(result.CheckIn).NotTo(BeNil())
				Expect(result.CheckIn.ID).To(Equal(checkinID))
			})
		})

		When("participant has not checked in", func() {
			It("should return not checked-in status", func() {
				participantID := uuid.New()

				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Not Checked In",
					Email:   "notcheckedin@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				mockEventRepo.events[testEventID] = event
				mockParticipant.participantsByID[participantID] = participant

				result, err := usecase.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.IsCheckedIn).To(BeFalse())
				Expect(result.CheckIn).To(BeNil())
			})
		})

		When("participant does not exist", func() {
			It("should return not found error", func() {
				participantID := uuid.New()

				result, err := usecase.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		When("user does not have permission", func() {
			It("should return forbidden error", func() {
				participantID := uuid.New()

				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Test User",
					Email:   "test@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testOrganizerID, // Different from testUserID
					Name:        "Test Event",
				}

				mockEventRepo.events[testEventID] = event
				mockParticipant.participantsByID[participantID] = participant

				result, err := usecase.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeForbidden))
			})
		})
	})

	Describe("List", func() {
		When("listing check-ins for an event", func() {
			Context("with multiple check-ins", func() {
				It("should return paginated list", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}

					participants := []*entity.Participant{
						{ID: uuid.New(), EventID: testEventID, Name: "User 1", Email: "user1@example.com"},
						{ID: uuid.New(), EventID: testEventID, Name: "User 2", Email: "user2@example.com"},
					}

					checkins := []*entity.Checkin{
						{
							ID:            uuid.New(),
							EventID:       testEventID,
							ParticipantID: participants[0].ID,
							CheckedInAt:   time.Now(),
							Method:        entity.CheckinMethodQRCode,
						},
						{
							ID:            uuid.New(),
							EventID:       testEventID,
							ParticipantID: participants[1].ID,
							CheckedInAt:   time.Now(),
							Method:        entity.CheckinMethodManual,
						},
					}

					mockEventRepo.events[testEventID] = event
					mockParticipant.participantsByID[participants[0].ID] = participants[0]
					mockParticipant.participantsByID[participants[1].ID] = participants[1]
					mockCheckinRepo.checkinsByEvent[testEventID] = checkins

					input := checkin.ListCheckInsInput{
						EventID: testEventID,
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.CheckIns).To(HaveLen(2))
					Expect(result.TotalCount).To(Equal(int64(2)))
				})
			})

			Context("with empty list", func() {
				It("should return empty list", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}

					mockEventRepo.events[testEventID] = event
					mockCheckinRepo.checkinsByEvent[testEventID] = []*entity.Checkin{}

					input := checkin.ListCheckInsInput{
						EventID: testEventID,
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.CheckIns).To(BeEmpty())
					Expect(result.TotalCount).To(Equal(int64(0)))
				})
			})

			Context("with pagination", func() {
				It("should respect page and per_page parameters", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}

					mockEventRepo.events[testEventID] = event

					input := checkin.ListCheckInsInput{
						EventID: testEventID,
						Page:    2,
						PerPage: 5,
					}

					_, err := usecase.List(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					// Verify offset calculation (page-1)*perPage = (2-1)*5 = 5
					Expect(mockCheckinRepo.lastOffset).To(Equal(5))
					Expect(mockCheckinRepo.lastLimit).To(Equal(5))
				})
			})
		})

		When("user does not have permission", func() {
			It("should return forbidden error", func() {
				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testOrganizerID, // Different from testUserID
					Name:        "Test Event",
				}

				mockEventRepo.events[testEventID] = event

				input := checkin.ListCheckInsInput{
					EventID: testEventID,
					Page:    1,
					PerPage: 10,
				}

				result, err := usecase.List(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeForbidden))
			})
		})

		When("event does not exist", func() {
			It("should return not found error", func() {
				input := checkin.ListCheckInsInput{
					EventID: testEventID,
					Page:    1,
					PerPage: 10,
				}

				result, err := usecase.List(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("Cancel", func() {
		When("canceling a check-in", func() {
			Context("as event organizer", func() {
				It("should successfully cancel check-in", func() {
					checkinID := uuid.New()
					checkinRecord := &entity.Checkin{
						ID:            checkinID,
						EventID:       testEventID,
						ParticipantID: uuid.New(),
						CheckedInAt:   time.Now(),
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}

					mockEventRepo.events[testEventID] = event
					mockCheckinRepo.checkinsByID[checkinID] = checkinRecord

					err := usecase.Cancel(ctx, testUserID, false, checkinID)

					Expect(err).NotTo(HaveOccurred())
					Expect(mockCheckinRepo.deleted).To(ContainElement(checkinID))
				})
			})

			Context("as admin", func() {
				It("should successfully cancel check-in", func() {
					checkinID := uuid.New()
					checkinRecord := &entity.Checkin{
						ID:            checkinID,
						EventID:       testEventID,
						ParticipantID: uuid.New(),
						CheckedInAt:   time.Now(),
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID, // Different from testUserID
						Name:        "Test Event",
					}

					mockEventRepo.events[testEventID] = event
					mockCheckinRepo.checkinsByID[checkinID] = checkinRecord

					err := usecase.Cancel(ctx, testUserID, true, checkinID) // isAdmin = true

					Expect(err).NotTo(HaveOccurred())
					Expect(mockCheckinRepo.deleted).To(ContainElement(checkinID))
				})
			})
		})

		When("check-in does not exist", func() {
			It("should return not found error", func() {
				checkinID := uuid.New()

				err := usecase.Cancel(ctx, testUserID, false, checkinID)

				Expect(err).To(HaveOccurred())
			})
		})

		When("user does not have permission", func() {
			It("should return forbidden error", func() {
				checkinID := uuid.New()
				checkinRecord := &entity.Checkin{
					ID:            checkinID,
					EventID:       testEventID,
					ParticipantID: uuid.New(),
					CheckedInAt:   time.Now(),
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testOrganizerID, // Different from testUserID
					Name:        "Test Event",
				}

				mockEventRepo.events[testEventID] = event
				mockCheckinRepo.checkinsByID[checkinID] = checkinRecord

				err := usecase.Cancel(ctx, testUserID, false, checkinID) // Not admin, not organizer

				Expect(err).To(HaveOccurred())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeForbidden))
			})
		})
	})
})

// Mock implementations

type mockCheckinRepository struct {
	checkinsByID          map[uuid.UUID]*entity.Checkin
	checkinsByParticipant map[uuid.UUID]*entity.Checkin
	checkinsByEvent       map[uuid.UUID][]*entity.Checkin
	existsMap             map[uuid.UUID]bool
	created               []*entity.Checkin
	deleted               []uuid.UUID
	lastLimit             int
	lastOffset            int
}

func newMockCheckinRepository() *mockCheckinRepository {
	return &mockCheckinRepository{
		checkinsByID:          make(map[uuid.UUID]*entity.Checkin),
		checkinsByParticipant: make(map[uuid.UUID]*entity.Checkin),
		checkinsByEvent:       make(map[uuid.UUID][]*entity.Checkin),
		existsMap:             make(map[uuid.UUID]bool),
		created:               []*entity.Checkin{},
		deleted:               []uuid.UUID{},
	}
}

func (m *mockCheckinRepository) Create(ctx context.Context, checkin *entity.Checkin) error {
	m.created = append(m.created, checkin)
	m.checkinsByID[checkin.ID] = checkin
	m.checkinsByParticipant[checkin.ParticipantID] = checkin
	return nil
}

func (m *mockCheckinRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Checkin, error) {
	if checkin, ok := m.checkinsByID[id]; ok {
		return checkin, nil
	}
	return nil, apperrors.NotFound("check-in not found")
}

func (m *mockCheckinRepository) FindByParticipant(ctx context.Context, participantID uuid.UUID) (*entity.Checkin, error) {
	if checkin, ok := m.checkinsByParticipant[participantID]; ok {
		return checkin, nil
	}
	return nil, apperrors.NotFound("check-in not found")
}

func (m *mockCheckinRepository) FindByEvent(ctx context.Context, eventID uuid.UUID, limit, offset int) ([]*entity.Checkin, int64, error) {
	m.lastLimit = limit
	m.lastOffset = offset
	checkins := m.checkinsByEvent[eventID]
	if checkins == nil {
		checkins = []*entity.Checkin{}
	}
	return checkins, int64(len(checkins)), nil
}

func (m *mockCheckinRepository) GetEventStats(ctx context.Context, eventID uuid.UUID) (*repository.CheckinStats, error) {
	return &repository.CheckinStats{}, nil
}

func (m *mockCheckinRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.checkinsByID[id]; !ok {
		return apperrors.NotFound("check-in not found")
	}
	m.deleted = append(m.deleted, id)
	delete(m.checkinsByID, id)
	return nil
}

func (m *mockCheckinRepository) ExistsByParticipant(ctx context.Context, eventID, participantID uuid.UUID) (bool, error) {
	exists, ok := m.existsMap[participantID]
	if !ok {
		return false, nil
	}
	return exists, nil
}

func (m *mockCheckinRepository) HealthCheck(ctx context.Context) error {
	return nil
}

type mockParticipantRepository struct {
	participants     map[string]*entity.Participant // key is QR code
	participantsByID map[uuid.UUID]*entity.Participant
}

func newMockParticipantRepository() *mockParticipantRepository {
	return &mockParticipantRepository{
		participants:     make(map[string]*entity.Participant),
		participantsByID: make(map[uuid.UUID]*entity.Participant),
	}
}

func (m *mockParticipantRepository) FindByQRCode(ctx context.Context, qrCode string) (*entity.Participant, error) {
	if p, ok := m.participants[qrCode]; ok {
		return p, nil
	}
	return nil, apperrors.NotFound("participant not found")
}

func (m *mockParticipantRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Participant, error) {
	if p, ok := m.participantsByID[id]; ok {
		return p, nil
	}
	return nil, apperrors.NotFound("participant not found")
}

func (m *mockParticipantRepository) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}

func (m *mockParticipantRepository) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}

func (m *mockParticipantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockParticipantRepository) ExistsByEmail(ctx context.Context, eventID uuid.UUID, email string) (bool, error) {
	return false, nil
}

func (m *mockParticipantRepository) BulkCreate(ctx context.Context, participants []*entity.Participant) error {
	return nil
}

func (m *mockParticipantRepository) FindByEventID(ctx context.Context, eventID uuid.UUID, offset, limit int) ([]*entity.Participant, int64, error) {
	return nil, 0, nil
}

func (m *mockParticipantRepository) Search(ctx context.Context, eventID uuid.UUID, query string, offset, limit int) ([]*entity.Participant, int64, error) {
	return nil, 0, nil
}

func (m *mockParticipantRepository) GetPaymentStats(ctx context.Context, eventID uuid.UUID) (*repository.ParticipantPaymentStats, error) {
	return &repository.ParticipantPaymentStats{}, nil
}

func (m *mockParticipantRepository) HealthCheck(ctx context.Context) error {
	return nil
}

type mockEventRepository struct {
	events map[uuid.UUID]*entity.Event
}

func newMockEventRepository() *mockEventRepository {
	return &mockEventRepository{
		events: make(map[uuid.UUID]*entity.Event),
	}
}

func (m *mockEventRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
	if event, ok := m.events[id]; ok {
		return event, nil
	}
	return nil, apperrors.NotFound("event not found")
}

func (m *mockEventRepository) Create(ctx context.Context, event *entity.Event) error {
	return nil
}

func (m *mockEventRepository) Update(ctx context.Context, event *entity.Event) error {
	return nil
}

func (m *mockEventRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockEventRepository) List(ctx context.Context, filter repository.EventListFilter, offset, limit int) ([]*entity.Event, int64, error) {
	return nil, 0, nil
}

func (m *mockEventRepository) GetStats(ctx context.Context, id uuid.UUID) (*repository.EventStats, error) {
	return &repository.EventStats{}, nil
}

func (m *mockEventRepository) HealthCheck(ctx context.Context) error {
	return nil
}
