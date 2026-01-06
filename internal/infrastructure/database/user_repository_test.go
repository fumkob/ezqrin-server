//go:build integration
// +build integration

package database_test

import (
	"context"
	"fmt"
	"time"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserRepository", func() {
	var (
		ctx        context.Context
		log        *logger.Logger
		cfg        *config.DatabaseConfig
		db         *database.PostgresDB
		repo       *database.UserRepository
		testUserID uuid.UUID
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

		repo = database.NewUserRepository(db.GetPool(), log).(*database.UserRepository)
		testUserID = uuid.New()
	})

	AfterEach(func() {
		// Clean up test data
		if db != nil {
			pool := db.GetPool()
			// Use TRUNCATE for complete cleanup
			_, _ = pool.Exec(ctx, "TRUNCATE TABLE users CASCADE")
			db.Close()
		}
	})

	When("creating a user", func() {
		Context("with valid data", func() {
			It("should create user successfully", func() {
				user := &entity.User{
					ID:           testUserID,
					Email:        "testuser@example.com",
					PasswordHash: "hashed_password_123",
					Name:         "Test User",
					Role:         entity.RoleOrganizer,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				err := repo.Create(ctx, user)

				Expect(err).To(BeNil())
			})

			It("should create admin user", func() {
				user := &entity.User{
					ID:           uuid.New(),
					Email:        "testadmin@example.com",
					PasswordHash: "hashed_password_admin",
					Name:         "Test Admin",
					Role:         entity.RoleAdmin,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				err := repo.Create(ctx, user)

				Expect(err).To(BeNil())
			})

			It("should create staff user", func() {
				user := &entity.User{
					ID:           uuid.New(),
					Email:        "teststaff@example.com",
					PasswordHash: "hashed_password_staff",
					Name:         "Test Staff",
					Role:         entity.RoleStaff,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				err := repo.Create(ctx, user)

				Expect(err).To(BeNil())
			})
		})

		Context("with duplicate email", func() {
			It("should return conflict error", func() {
				user1 := &entity.User{
					ID:           uuid.New(),
					Email:        "duplicate@example.com",
					PasswordHash: "hashed_password_1",
					Name:         "User One",
					Role:         entity.RoleOrganizer,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				err := repo.Create(ctx, user1)
				Expect(err).To(BeNil())

				user2 := &entity.User{
					ID:           uuid.New(),
					Email:        "duplicate@example.com", // Same email
					PasswordHash: "hashed_password_2",
					Name:         "User Two",
					Role:         entity.RoleOrganizer,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				err = repo.Create(ctx, user2)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsConflict(err)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})
		})

	})

	When("finding user by ID", func() {
		var createdUser *entity.User

		BeforeEach(func() {
			createdUser = &entity.User{
				ID:           uuid.New(),
				Email:        "testfindbyid@example.com",
				PasswordHash: "hashed_password_findbyid",
				Name:         "Find By ID User",
				Role:         entity.RoleOrganizer,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			err := repo.Create(ctx, createdUser)
			Expect(err).To(BeNil())
		})

		Context("with existing ID", func() {
			It("should return user without password hash", func() {
				found, err := repo.FindByID(ctx, createdUser.ID)

				Expect(err).To(BeNil())
				Expect(found).NotTo(BeNil())
				Expect(found.ID).To(Equal(createdUser.ID))
				Expect(found.Email).To(Equal(createdUser.Email))
				Expect(found.Name).To(Equal(createdUser.Name))
				Expect(found.Role).To(Equal(createdUser.Role))
				Expect(found.PasswordHash).To(BeEmpty()) // IMPORTANT: Should NOT return password hash
				Expect(found.IsAnonymized).To(BeFalse())
			})
		})

		Context("with non-existent ID", func() {
			It("should return not found error", func() {
				nonExistentID := uuid.New()

				found, err := repo.FindByID(ctx, nonExistentID)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("not found"))
				Expect(found).To(BeNil())
			})
		})
	})

	When("finding user by email", func() {
		var createdUser *entity.User

		BeforeEach(func() {
			createdUser = &entity.User{
				ID:           uuid.New(),
				Email:        "testfindbyemail@example.com",
				PasswordHash: "hashed_password_findbyemail",
				Name:         "Find By Email User",
				Role:         entity.RoleOrganizer,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			err := repo.Create(ctx, createdUser)
			Expect(err).To(BeNil())
		})

		Context("with existing email", func() {
			It("should return user without password hash", func() {
				found, err := repo.FindByEmail(ctx, createdUser.Email)

				Expect(err).To(BeNil())
				Expect(found).NotTo(BeNil())
				Expect(found.ID).To(Equal(createdUser.ID))
				Expect(found.Email).To(Equal(createdUser.Email))
				Expect(found.Name).To(Equal(createdUser.Name))
				Expect(found.Role).To(Equal(createdUser.Role))
				Expect(found.PasswordHash).To(BeEmpty()) // IMPORTANT: Should NOT return password hash
			})
		})

		Context("with non-existent email", func() {
			It("should return not found error", func() {
				found, err := repo.FindByEmail(ctx, "nonexistent@example.com")

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("not found"))
				Expect(found).To(BeNil())
			})
		})

		Context("with FindByEmailWithPassword", func() {
			It("should return user with password hash for authentication", func() {
				found, err := repo.FindByEmailWithPassword(ctx, createdUser.Email)

				Expect(err).To(BeNil())
				Expect(found).NotTo(BeNil())
				Expect(found.ID).To(Equal(createdUser.ID))
				Expect(found.Email).To(Equal(createdUser.Email))
				Expect(found.Name).To(Equal(createdUser.Name))
				Expect(found.Role).To(Equal(createdUser.Role))
				Expect(found.PasswordHash).To(Equal(createdUser.PasswordHash)) // IMPORTANT: Should return password hash
			})

			It("should return not found error for non-existent email", func() {
				found, err := repo.FindByEmailWithPassword(ctx, "nonexistent@example.com")

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(found).To(BeNil())
			})
		})
	})

	When("updating user", func() {
		var createdUser *entity.User

		BeforeEach(func() {
			createdUser = &entity.User{
				ID:           uuid.New(),
				Email:        "testupdate@example.com",
				PasswordHash: "hashed_password_update",
				Name:         "Update User",
				Role:         entity.RoleOrganizer,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			err := repo.Create(ctx, createdUser)
			Expect(err).To(BeNil())
		})

		Context("with valid changes", func() {
			It("should update name successfully", func() {
				createdUser.Name = "Updated Name"
				createdUser.UpdatedAt = time.Now()

				err := repo.Update(ctx, createdUser)

				Expect(err).To(BeNil())

				// Verify update
				found, err := repo.FindByID(ctx, createdUser.ID)
				Expect(err).To(BeNil())
				Expect(found.Name).To(Equal("Updated Name"))
			})

			It("should update email successfully", func() {
				createdUser.Email = "testupdated@example.com"
				createdUser.UpdatedAt = time.Now()

				err := repo.Update(ctx, createdUser)

				Expect(err).To(BeNil())

				// Verify update
				found, err := repo.FindByID(ctx, createdUser.ID)
				Expect(err).To(BeNil())
				Expect(found.Email).To(Equal("testupdated@example.com"))
			})

			It("should update role successfully", func() {
				createdUser.Role = entity.RoleAdmin
				createdUser.UpdatedAt = time.Now()

				err := repo.Update(ctx, createdUser)

				Expect(err).To(BeNil())

				// Verify update
				found, err := repo.FindByID(ctx, createdUser.ID)
				Expect(err).To(BeNil())
				Expect(found.Role).To(Equal(entity.RoleAdmin))
			})

			It("should update password hash successfully", func() {
				newPasswordHash := "new_hashed_password"
				createdUser.PasswordHash = newPasswordHash
				createdUser.UpdatedAt = time.Now()

				err := repo.Update(ctx, createdUser)

				Expect(err).To(BeNil())

				// Verify update via FindByEmailWithPassword
				found, err := repo.FindByEmailWithPassword(ctx, createdUser.Email)
				Expect(err).To(BeNil())
				Expect(found.PasswordHash).To(Equal(newPasswordHash))
			})
		})

		Context("with duplicate email", func() {
			It("should return conflict error", func() {
				// Create another user
				anotherUser := &entity.User{
					ID:           uuid.New(),
					Email:        "testanother@example.com",
					PasswordHash: "hashed_password",
					Name:         "Another User",
					Role:         entity.RoleOrganizer,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err := repo.Create(ctx, anotherUser)
				Expect(err).To(BeNil())

				// Try to update createdUser with anotherUser's email
				createdUser.Email = "testanother@example.com"
				createdUser.UpdatedAt = time.Now()

				err = repo.Update(ctx, createdUser)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsConflict(err)).To(BeTrue())
			})
		})

		Context("with non-existent user ID", func() {
			It("should return not found error", func() {
				nonExistentUser := &entity.User{
					ID:           uuid.New(),
					Email:        "testnonexistent@example.com",
					PasswordHash: "hashed_password",
					Name:         "Non-existent User",
					Role:         entity.RoleOrganizer,
					UpdatedAt:    time.Now(),
				}

				err := repo.Update(ctx, nonExistentUser)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
			})
		})
	})

	When("listing users", func() {
		BeforeEach(func() {
			// Create multiple test users
			for i := 1; i <= 5; i++ {
				user := &entity.User{
					ID:           uuid.New(),
					Email:        fmt.Sprintf("testlist%d@example.com", i),
					PasswordHash: fmt.Sprintf("hashed_password_%d", i),
					Name:         fmt.Sprintf("List User %d", i),
					Role:         entity.RoleOrganizer,
					CreatedAt:    time.Now().Add(-time.Duration(5-i) * time.Minute), // Stagger creation times
					UpdatedAt:    time.Now(),
				}
				err := repo.Create(ctx, user)
				Expect(err).To(BeNil())
			}
		})

		Context("with pagination", func() {
			It("should return paginated results with total count", func() {
				users, total, err := repo.List(ctx, 0, 3)

				Expect(err).To(BeNil())
				Expect(total).To(BeNumerically(">=", 5))
				Expect(users).To(HaveLen(3))
				// Verify password hash is not returned
				for _, user := range users {
					Expect(user.PasswordHash).To(BeEmpty())
				}
			})

			It("should return second page correctly", func() {
				users, total, err := repo.List(ctx, 3, 2)

				Expect(err).To(BeNil())
				Expect(total).To(BeNumerically(">=", 5))
				Expect(users).To(HaveLen(2))
			})

			It("should return empty list beyond available pages", func() {
				users, total, err := repo.List(ctx, 100, 10)

				Expect(err).To(BeNil())
				Expect(total).To(BeNumerically(">=", 5))
				Expect(users).To(BeEmpty())
			})
		})

		Context("with soft-deleted users", func() {
			It("should exclude soft-deleted users from list", func() {
				// Create and soft delete a user
				userToDelete := &entity.User{
					ID:           uuid.New(),
					Email:        "testdeleted@example.com",
					PasswordHash: "hashed_password",
					Name:         "Deleted User",
					Role:         entity.RoleOrganizer,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err := repo.Create(ctx, userToDelete)
				Expect(err).To(BeNil())

				// Create deleter user first
				deleterUser := &entity.User{
					ID:           uuid.New(),
					Email:        "testlistdeleter@example.com",
					PasswordHash: "hashed_password_admin",
					Name:         "Deleter User",
					Role:         entity.RoleAdmin,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err = repo.Create(ctx, deleterUser)
				Expect(err).To(BeNil())

				err = repo.SoftDelete(ctx, userToDelete.ID, deleterUser.ID)
				Expect(err).To(BeNil())

				// List should not include deleted user
				users, _, err := repo.List(ctx, 0, 100)
				Expect(err).To(BeNil())

				for _, user := range users {
					Expect(user.ID).NotTo(Equal(userToDelete.ID))
					Expect(user.IsAnonymized).To(BeFalse())
				}
			})
		})

		Context("with ordering", func() {
			It("should return users ordered by created_at DESC", func() {
				users, _, err := repo.List(ctx, 0, 10)

				Expect(err).To(BeNil())
				Expect(users).NotTo(BeEmpty())

				// Verify ordering (most recent first)
				for i := 1; i < len(users); i++ {
					Expect(users[i-1].CreatedAt.After(users[i].CreatedAt) ||
						users[i-1].CreatedAt.Equal(users[i].CreatedAt)).To(BeTrue())
				}
			})
		})
	})

	When("soft deleting user", func() {
		var createdUser *entity.User
		var deletedBy uuid.UUID
		var deleterUser *entity.User

		BeforeEach(func() {
			// Create deleter user first (for foreign key constraint)
			deleterUser = &entity.User{
				ID:           uuid.New(),
				Email:        "testdeleter@example.com",
				PasswordHash: "hashed_password_deleter",
				Name:         "Deleter User",
				Role:         entity.RoleAdmin,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			err := repo.Create(ctx, deleterUser)
			Expect(err).To(BeNil())
			deletedBy = deleterUser.ID

			// Create user to be deleted
			createdUser = &entity.User{
				ID:           uuid.New(),
				Email:        "testsoftdelete@example.com",
				PasswordHash: "hashed_password_delete",
				Name:         "Soft Delete User",
				Role:         entity.RoleOrganizer,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			err = repo.Create(ctx, createdUser)
			Expect(err).To(BeNil())
		})

		Context("with valid user ID", func() {
			It("should anonymize PII and set deleted_at", func() {
				err := repo.SoftDelete(ctx, createdUser.ID, deletedBy)

				Expect(err).To(BeNil())

				// Verify anonymization
				found, err := repo.FindByID(ctx, createdUser.ID)
				Expect(err).To(BeNil())
				Expect(found.Email).To(Equal(fmt.Sprintf("deleted_%s@anonymized.local", createdUser.ID.String())))
				Expect(found.Name).To(Equal("Deleted User"))
				Expect(found.DeletedAt).NotTo(BeNil())
				Expect(found.DeletedBy).NotTo(BeNil())
				Expect(*found.DeletedBy).To(Equal(deletedBy))
				Expect(found.IsAnonymized).To(BeTrue())
			})

			It("should not return original email after anonymization", func() {
				originalEmail := createdUser.Email

				err := repo.SoftDelete(ctx, createdUser.ID, deletedBy)
				Expect(err).To(BeNil())

				// Try to find by original email
				_, err = repo.FindByEmail(ctx, originalEmail)
				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("with non-existent user ID", func() {
			It("should return not found error", func() {
				nonExistentID := uuid.New()

				err := repo.SoftDelete(ctx, nonExistentID, deletedBy)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("not found or already deleted"))
			})
		})

		Context("with already deleted user", func() {
			It("should return not found error", func() {
				// First deletion
				err := repo.SoftDelete(ctx, createdUser.ID, deletedBy)
				Expect(err).To(BeNil())

				// Second deletion attempt
				err = repo.SoftDelete(ctx, createdUser.ID, deletedBy)

				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("not found or already deleted"))
			})
		})
	})

	When("checking email existence", func() {
		var createdUser *entity.User

		BeforeEach(func() {
			createdUser = &entity.User{
				ID:           uuid.New(),
				Email:        "testexists@example.com",
				PasswordHash: "hashed_password_exists",
				Name:         "Exists User",
				Role:         entity.RoleOrganizer,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			err := repo.Create(ctx, createdUser)
			Expect(err).To(BeNil())
		})

		Context("with existing email", func() {
			It("should return true", func() {
				exists, err := repo.ExistsByEmail(ctx, createdUser.Email)

				Expect(err).To(BeNil())
				Expect(exists).To(BeTrue())
			})
		})

		Context("with non-existent email", func() {
			It("should return false", func() {
				exists, err := repo.ExistsByEmail(ctx, "nonexistent@example.com")

				Expect(err).To(BeNil())
				Expect(exists).To(BeFalse())
			})
		})

		Context("with soft-deleted user email", func() {
			It("should return true for anonymized email", func() {
				// Create deleter user first
				deleterUser := &entity.User{
					ID:           uuid.New(),
					Email:        "testexistsdeleter@example.com",
					PasswordHash: "hashed_password_deleter",
					Name:         "Deleter User",
					Role:         entity.RoleAdmin,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err := repo.Create(ctx, deleterUser)
				Expect(err).To(BeNil())

				err = repo.SoftDelete(ctx, createdUser.ID, deleterUser.ID)
				Expect(err).To(BeNil())

				// Should not exist with original email
				exists, err := repo.ExistsByEmail(ctx, createdUser.Email)
				Expect(err).To(BeNil())
				Expect(exists).To(BeFalse())

				// Should exist with anonymized email
				anonymizedEmail := fmt.Sprintf("deleted_%s@anonymized.local", createdUser.ID.String())
				exists, err = repo.ExistsByEmail(ctx, anonymizedEmail)
				Expect(err).To(BeNil())
				Expect(exists).To(BeTrue())
			})
		})
	})

	When("performing health check", func() {
		Context("with active connection", func() {
			It("should return no error", func() {
				err := repo.HealthCheck(ctx)

				Expect(err).To(BeNil())
			})
		})
	})
})
