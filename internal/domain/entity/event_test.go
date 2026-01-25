package entity_test

import (
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event", func() {
	var (
		validEvent *entity.Event
		now        time.Time
	)

	BeforeEach(func() {
		now = time.Now()
		validEvent = &entity.Event{
			ID:          uuid.New(),
			OrganizerID: uuid.New(),
			Name:        "Tech Conference 2025",
			Description: "Annual tech conference",
			StartDate:   now.Add(24 * time.Hour),
			Location:    "Tokyo",
			Timezone:    "Asia/Tokyo",
			Status:      entity.StatusDraft,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	})

	When("validating event", func() {
		Context("with all required fields", func() {
			It("should succeed", func() {
				Expect(validEvent.Validate()).To(Succeed())
			})
		})

		Context("with empty name", func() {
			It("should fail", func() {
				validEvent.Name = ""
				Expect(validEvent.Validate()).To(MatchError(entity.ErrEventNameRequired))
			})
		})

		Context("with name too long", func() {
			It("should fail", func() {
				validEvent.Name = string(make([]byte, entity.EventNameMaxLength+1))
				Expect(validEvent.Validate()).To(MatchError(entity.ErrEventNameTooLong))
			})
		})

		Context("with description too long", func() {
			It("should fail", func() {
				validEvent.Description = string(make([]byte, entity.EventDescriptionMaxLength+1))
				Expect(validEvent.Validate()).To(MatchError(entity.ErrEventDescriptionTooLong))
			})
		})

		Context("with zero start date", func() {
			It("should fail", func() {
				validEvent.StartDate = time.Time{}
				Expect(validEvent.Validate()).To(MatchError(entity.ErrEventStartDateRequired))
			})
		})

		Context("with end date before start date", func() {
			It("should fail", func() {
				endDate := validEvent.StartDate.Add(-1 * time.Hour)
				validEvent.EndDate = &endDate
				Expect(validEvent.Validate()).To(MatchError(entity.ErrEventEndDateBeforeStart))
			})
		})

		Context("with end date after start date", func() {
			It("should succeed", func() {
				endDate := validEvent.StartDate.Add(1 * time.Hour)
				validEvent.EndDate = &endDate
				Expect(validEvent.Validate()).To(Succeed())
			})
		})

		Context("with location too long", func() {
			It("should fail", func() {
				validEvent.Location = string(make([]byte, entity.EventLocationMaxLength+1))
				Expect(validEvent.Validate()).To(MatchError(entity.ErrEventLocationTooLong))
			})
		})

		Context("with invalid status", func() {
			It("should fail", func() {
				validEvent.Status = "invalid"
				Expect(validEvent.Validate()).To(MatchError(entity.ErrEventStatusInvalid))
			})
		})
	})

	When("transitioning event status", func() {
		Context("from StatusDraft", func() {
			BeforeEach(func() {
				validEvent.Status = entity.StatusDraft
			})

			It("should be able to transition to StatusPublished", func() {
				Expect(validEvent.CanTransitionTo(entity.StatusPublished)).To(BeTrue())
				Expect(validEvent.TransitionTo(entity.StatusPublished)).To(Succeed())
				Expect(validEvent.Status).To(Equal(entity.StatusPublished))
			})

			It("should be able to transition to StatusCancelled", func() {
				Expect(validEvent.CanTransitionTo(entity.StatusCancelled)).To(BeTrue())
				Expect(validEvent.TransitionTo(entity.StatusCancelled)).To(Succeed())
				Expect(validEvent.Status).To(Equal(entity.StatusCancelled))
			})

			It("should not be able to transition to StatusOngoing", func() {
				Expect(validEvent.CanTransitionTo(entity.StatusOngoing)).To(BeFalse())
				Expect(validEvent.TransitionTo(entity.StatusOngoing)).To(MatchError(entity.ErrEventInvalidTransition))
			})
		})

		Context("from StatusPublished", func() {
			BeforeEach(func() {
				validEvent.Status = entity.StatusPublished
			})

			It("should be able to transition to StatusOngoing", func() {
				Expect(validEvent.CanTransitionTo(entity.StatusOngoing)).To(BeTrue())
				Expect(validEvent.TransitionTo(entity.StatusOngoing)).To(Succeed())
			})

			It("should be able to transition to StatusCancelled", func() {
				Expect(validEvent.CanTransitionTo(entity.StatusCancelled)).To(BeTrue())
				Expect(validEvent.TransitionTo(entity.StatusCancelled)).To(Succeed())
			})

			It("should not be able to transition back to StatusDraft", func() {
				Expect(validEvent.CanTransitionTo(entity.StatusDraft)).To(BeFalse())
				Expect(validEvent.TransitionTo(entity.StatusDraft)).To(MatchError(entity.ErrEventInvalidTransition))
			})
		})

		Context("from StatusOngoing", func() {
			BeforeEach(func() {
				validEvent.Status = entity.StatusOngoing
			})

			It("should be able to transition to StatusCompleted", func() {
				Expect(validEvent.CanTransitionTo(entity.StatusCompleted)).To(BeTrue())
				Expect(validEvent.TransitionTo(entity.StatusCompleted)).To(Succeed())
			})

			It("should be able to transition to StatusCancelled", func() {
				Expect(validEvent.CanTransitionTo(entity.StatusCancelled)).To(BeTrue())
				Expect(validEvent.TransitionTo(entity.StatusCancelled)).To(Succeed())
			})
		})

		Context("with terminal statuses", func() {
			It("should not be able to transition from StatusCompleted", func() {
				validEvent.Status = entity.StatusCompleted
				Expect(validEvent.CanTransitionTo(entity.StatusDraft)).To(BeFalse())
				Expect(validEvent.TransitionTo(entity.StatusPublished)).To(MatchError(entity.ErrEventInvalidTransition))
			})

			It("should not be able to transition from StatusCancelled", func() {
				validEvent.Status = entity.StatusCancelled
				Expect(validEvent.CanTransitionTo(entity.StatusDraft)).To(BeFalse())
				Expect(validEvent.TransitionTo(entity.StatusPublished)).To(MatchError(entity.ErrEventInvalidTransition))
			})
		})
	})
})
