package entity_test

import (
	"testing"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User Entity Suite")
}

var _ = Describe("User", func() {
	Describe("Validate", func() {
		When("validating a user", func() {
			Context("with all valid fields", func() {
				It("should validate successfully", func() {
					user := &entity.User{
						ID:           uuid.New(),
						Email:        "test@example.com",
						PasswordHash: "hashed_password",
						Name:         "John Doe",
						Role:         entity.RoleOrganizer,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}

					err := user.Validate()
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with missing email", func() {
				It("should return email required error", func() {
					user := &entity.User{
						ID:           uuid.New(),
						Email:        "",
						PasswordHash: "hashed_password",
						Name:         "John Doe",
						Role:         entity.RoleOrganizer,
					}

					err := user.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(entity.ErrUserEmailRequired))
				})
			})

			Context("with invalid email format", func() {
				It("should return email invalid error", func() {
					user := &entity.User{
						ID:           uuid.New(),
						Email:        "invalid-email",
						PasswordHash: "hashed_password",
						Name:         "John Doe",
						Role:         entity.RoleOrganizer,
					}

					err := user.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(entity.ErrUserEmailInvalid))
				})
			})

			Context("with missing password hash", func() {
				It("should return password required error", func() {
					user := &entity.User{
						ID:           uuid.New(),
						Email:        "test@example.com",
						PasswordHash: "",
						Name:         "John Doe",
						Role:         entity.RoleOrganizer,
					}

					err := user.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(entity.ErrUserPasswordRequired))
				})
			})

			Context("with missing name", func() {
				It("should return name required error", func() {
					user := &entity.User{
						ID:           uuid.New(),
						Email:        "test@example.com",
						PasswordHash: "hashed_password",
						Name:         "",
						Role:         entity.RoleOrganizer,
					}

					err := user.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(entity.ErrUserNameRequired))
				})
			})

			Context("with name too short", func() {
				It("should return name too short error", func() {
					user := &entity.User{
						ID:           uuid.New(),
						Email:        "test@example.com",
						PasswordHash: "hashed_password",
						Name:         "A",
						Role:         entity.RoleOrganizer,
					}

					err := user.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(entity.ErrUserNameTooShort))
				})
			})

			Context("with name too long", func() {
				It("should return name too long error", func() {
					user := &entity.User{
						ID:           uuid.New(),
						Email:        "test@example.com",
						PasswordHash: "hashed_password",
						Name:         string(make([]byte, 256)),
						Role:         entity.RoleOrganizer,
					}

					err := user.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(entity.ErrUserNameTooLong))
				})
			})

			Context("with missing role", func() {
				It("should return role required error", func() {
					user := &entity.User{
						ID:           uuid.New(),
						Email:        "test@example.com",
						PasswordHash: "hashed_password",
						Name:         "John Doe",
						Role:         "",
					}

					err := user.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(entity.ErrUserRoleRequired))
				})
			})

			Context("with invalid role", func() {
				It("should return role invalid error", func() {
					user := &entity.User{
						ID:           uuid.New(),
						Email:        "test@example.com",
						PasswordHash: "hashed_password",
						Name:         "John Doe",
						Role:         "invalid",
					}

					err := user.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(entity.ErrUserRoleInvalid))
				})
			})
		})
	})

	Describe("IsDeleted", func() {
		When("checking if user is deleted", func() {
			Context("with nil DeletedAt", func() {
				It("should return false", func() {
					user := &entity.User{
						DeletedAt: nil,
					}

					Expect(user.IsDeleted()).To(BeFalse())
				})
			})

			Context("with DeletedAt set", func() {
				It("should return true", func() {
					now := time.Now()
					user := &entity.User{
						DeletedAt: &now,
					}

					Expect(user.IsDeleted()).To(BeTrue())
				})
			})
		})
	})

	Describe("Role Checks", func() {
		When("checking user role permissions", func() {
			Context("with admin role", func() {
				It("should identify as admin and allow event management", func() {
					user := &entity.User{Role: entity.RoleAdmin}

					Expect(user.IsAdmin()).To(BeTrue())
					Expect(user.IsOrganizer()).To(BeFalse())
					Expect(user.IsStaff()).To(BeFalse())
					Expect(user.CanManageEvents()).To(BeTrue())
				})
			})

			Context("with organizer role", func() {
				It("should identify as organizer and allow event management", func() {
					user := &entity.User{Role: entity.RoleOrganizer}

					Expect(user.IsAdmin()).To(BeFalse())
					Expect(user.IsOrganizer()).To(BeTrue())
					Expect(user.IsStaff()).To(BeFalse())
					Expect(user.CanManageEvents()).To(BeTrue())
				})
			})

			Context("with staff role", func() {
				It("should identify as staff without event management permission", func() {
					user := &entity.User{Role: entity.RoleStaff}

					Expect(user.IsAdmin()).To(BeFalse())
					Expect(user.IsOrganizer()).To(BeFalse())
					Expect(user.IsStaff()).To(BeTrue())
					Expect(user.CanManageEvents()).To(BeFalse())
				})
			})
		})
	})

	Describe("ValidateRole", func() {
		When("validating a role string", func() {
			Context("with admin role", func() {
				It("should validate successfully", func() {
					err := entity.ValidateRole("admin")
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with organizer role", func() {
				It("should validate successfully", func() {
					err := entity.ValidateRole("organizer")
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with staff role", func() {
				It("should validate successfully", func() {
					err := entity.ValidateRole("staff")
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with invalid role", func() {
				It("should return role invalid error", func() {
					err := entity.ValidateRole("invalid")
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(entity.ErrUserRoleInvalid))
				})
			})
		})
	})

	Describe("IsValidRole", func() {
		When("checking if user has a valid role", func() {
			Context("with admin role", func() {
				It("should return true", func() {
					user := &entity.User{Role: entity.RoleAdmin}
					Expect(user.IsValidRole()).To(BeTrue())
				})
			})

			Context("with organizer role", func() {
				It("should return true", func() {
					user := &entity.User{Role: entity.RoleOrganizer}
					Expect(user.IsValidRole()).To(BeTrue())
				})
			})

			Context("with staff role", func() {
				It("should return true", func() {
					user := &entity.User{Role: entity.RoleStaff}
					Expect(user.IsValidRole()).To(BeTrue())
				})
			})

			Context("with invalid role", func() {
				It("should return false", func() {
					user := &entity.User{Role: "invalid"}
					Expect(user.IsValidRole()).To(BeFalse())
				})
			})

			Context("with empty role", func() {
				It("should return false", func() {
					user := &entity.User{Role: ""}
					Expect(user.IsValidRole()).To(BeFalse())
				})
			})
		})
	})
})
