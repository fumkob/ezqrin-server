//go:build integration
// +build integration

package database_test

import (
	"context"
	"fmt"
	"time"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventRepository", func() {
	var (
		ctx         context.Context
		log         *logger.Logger
		cfg         *config.DatabaseConfig
		db          *database.PostgresDB
		repo        repository.EventRepository
		userRepo    repository.UserRepository
		testUserID  uuid.UUID
		testEventID uuid.UUID
		organizer   *entity.User
	)

	BeforeEach(func() {
		ctx = context.Background()
		log, _ = logger.New(logger.Config{
			Level:       "info",
			Format:      "console",
			Environment: "development",
		})
		cfg = &config.DatabaseConfig{
			Host:            "postgres",
			Port:            5432,
			User:            "ezqrin",
			Password:        "ezqrin_dev",
			Name:            "ezqrin_test",
			SSLMode:         "disable",
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		}

		var err error
		db, err = database.NewPostgresDB(ctx, cfg, log)
		Expect(err).To(BeNil())

		repo = database.NewEventRepository(db.GetPool(), log)
		userRepo = database.NewUserRepository(db.GetPool(), log)

		// Create an organizer for the events
		testUserID = uuid.New()
		organizer = &entity.User{
			ID:           testUserID,
			Email:        fmt.Sprintf("organizer_%s@example.com", testUserID.String()[:8]),
			PasswordHash: "hashed_password",
			Name:         "Organizer User",
			Role:         entity.RoleOrganizer,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		Expect(userRepo.Create(ctx, organizer)).To(Succeed())

		testEventID = uuid.New()
	})

	AfterEach(func() {
		if db != nil {
			pool := db.GetPool()
			_, _ = pool.Exec(ctx, "TRUNCATE TABLE users CASCADE")
			_, _ = pool.Exec(ctx, "TRUNCATE TABLE events CASCADE")
			db.Close()
		}
	})

	createTestEvent := func(id uuid.UUID, name string, organizerID uuid.UUID) *entity.Event {
		return &entity.Event{
			ID:          id,
			OrganizerID: organizerID,
			Name:        name,
			Description: "Test event description",
			StartDate:   time.Now().Add(24 * time.Hour),
			Location:    "Tokyo",
			Timezone:    "Asia/Tokyo",
			Status:      entity.StatusDraft,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	When("creating an event", func() {
		It("should create event successfully", func() {
			event := createTestEvent(testEventID, "New Event", testUserID)
			err := repo.Create(ctx, event)
			Expect(err).To(BeNil())

			found, err := repo.FindByID(ctx, testEventID)
			Expect(err).To(BeNil())
			Expect(found.Name).To(Equal("New Event"))
			Expect(found.OrganizerID).To(Equal(testUserID))
		})

		It("should return error if organizer does not exist", func() {
			event := createTestEvent(testEventID, "New Event", uuid.New())
			err := repo.Create(ctx, event)
			Expect(err).NotTo(BeNil())
		})
	})

	When("finding an event by ID", func() {
		BeforeEach(func() {
			event := createTestEvent(testEventID, "Find Me", testUserID)
			Expect(repo.Create(ctx, event)).To(Succeed())
		})

		It("should find event by ID", func() {
			found, err := repo.FindByID(ctx, testEventID)
			Expect(err).To(BeNil())
			Expect(found.ID).To(Equal(testEventID))
			Expect(found.Name).To(Equal("Find Me"))
		})

		It("should return not found error for non-existent ID", func() {
			found, err := repo.FindByID(ctx, uuid.New())
			Expect(err).NotTo(BeNil())
			Expect(apperrors.IsNotFound(err)).To(BeTrue())
			Expect(found).To(BeNil())
		})
	})

	When("listing events", func() {
		BeforeEach(func() {
			for i := 1; i <= 5; i++ {
				status := entity.StatusDraft
				if i%2 == 0 {
					status = entity.StatusPublished
				}
				event := createTestEvent(uuid.New(), fmt.Sprintf("Event %d", i), testUserID)
				event.Status = status
				event.CreatedAt = time.Now().Add(-time.Duration(10-i) * time.Hour)
				Expect(repo.Create(ctx, event)).To(Succeed())
			}
		})

		It("should list all events for an organizer", func() {
			filter := repository.EventListFilter{OrganizerID: &testUserID}
			events, total, err := repo.List(ctx, filter, 0, 10)
			Expect(err).To(BeNil())
			Expect(total).To(Equal(int64(5)))
			Expect(events).To(HaveLen(5))
		})

		It("should filter events by status", func() {
			status := entity.StatusPublished
			filter := repository.EventListFilter{Status: &status}
			events, total, err := repo.List(ctx, filter, 0, 10)
			Expect(err).To(BeNil())
			Expect(total).To(Equal(int64(2)))
			Expect(events).To(HaveLen(2))
		})

		It("should search events by name", func() {
			filter := repository.EventListFilter{Search: "Event 3"}
			events, total, err := repo.List(ctx, filter, 0, 10)
			Expect(err).To(BeNil())
			Expect(total).To(Equal(int64(1)))
			Expect(events[0].Name).To(Equal("Event 3"))
		})

		It("should handle pagination correctly", func() {
			filter := repository.EventListFilter{}
			events, total, err := repo.List(ctx, filter, 0, 3)
			Expect(err).To(BeNil())
			Expect(total).To(Equal(int64(5)))
			Expect(events).To(HaveLen(3))

			events2, total2, err := repo.List(ctx, filter, 3, 3)
			Expect(err).To(BeNil())
			Expect(total2).To(Equal(int64(5)))
			Expect(events2).To(HaveLen(2))
		})
	})

	When("updating an event", func() {
		BeforeEach(func() {
			event := createTestEvent(testEventID, "Original Name", testUserID)
			Expect(repo.Create(ctx, event)).To(Succeed())
		})

		It("should update event successfully", func() {
			found, _ := repo.FindByID(ctx, testEventID)
			found.Name = "Updated Name"
			found.Status = entity.StatusPublished
			found.UpdatedAt = time.Now()

			err := repo.Update(ctx, found)
			Expect(err).To(BeNil())

			updated, _ := repo.FindByID(ctx, testEventID)
			Expect(updated.Name).To(Equal("Updated Name"))
			Expect(updated.Status).To(Equal(entity.StatusPublished))
		})

		It("should return not found error for non-existent event", func() {
			event := createTestEvent(uuid.New(), "Non-existent", testUserID)
			err := repo.Update(ctx, event)
			Expect(err).NotTo(BeNil())
			Expect(apperrors.IsNotFound(err)).To(BeTrue())
		})
	})

	When("deleting an event", func() {
		BeforeEach(func() {
			event := createTestEvent(testEventID, "Delete Me", testUserID)
			Expect(repo.Create(ctx, event)).To(Succeed())
		})

		It("should delete event successfully", func() {
			err := repo.Delete(ctx, testEventID)
			Expect(err).To(BeNil())

			_, err = repo.FindByID(ctx, testEventID)
			Expect(apperrors.IsNotFound(err)).To(BeTrue())
		})

		It("should return not found error for non-existent event", func() {
			err := repo.Delete(ctx, uuid.New())
			Expect(err).NotTo(BeNil())
			Expect(apperrors.IsNotFound(err)).To(BeTrue())
		})
	})

	When("getting event stats", func() {
		BeforeEach(func() {
			event := createTestEvent(testEventID, "Stats Event", testUserID)
			Expect(repo.Create(ctx, event)).To(Succeed())
		})

		It("should return stats for an event", func() {
			stats, err := repo.GetStats(ctx, testEventID)
			Expect(err).To(BeNil())
			Expect(stats.TotalParticipants).To(Equal(int64(0)))
			Expect(stats.CheckedInCount).To(Equal(int64(0)))
		})

		It("should return not found error for non-existent event", func() {
			stats, err := repo.GetStats(ctx, uuid.New())
			Expect(err).NotTo(BeNil())
			Expect(apperrors.IsNotFound(err)).To(BeTrue())
			Expect(stats).To(BeNil())
		})
	})
})
