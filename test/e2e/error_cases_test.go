//go:build integration
// +build integration

// test/e2e/error_cases_test.go
package e2e_test

import (
	"net/http"

	"github.com/fumkob/ezqrin-server/test/fixtures"
	"github.com/fumkob/ezqrin-server/test/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error Scenarios", func() {
	var (
		env    *testutil.TestEnv
		helper *fixtures.Helper
	)

	BeforeEach(func() {
		var err error
		env, err = testutil.NewTestEnv()
		Expect(err).NotTo(HaveOccurred())
		testutil.CleanDatabase(env.DB, env.RedisClient)
		helper = fixtures.NewHelper(env.Router)
	})

	AfterEach(func() {
		if env != nil {
			testutil.CleanDatabase(env.DB, env.RedisClient)
		}
	})

	When("accessing protected endpoints without authentication", func() {
		Context("event management endpoints", func() {
			It("should return 401 for POST /api/v1/events without token", func() {
				w := helper.DoRequest(http.MethodPost, "/api/v1/events", "", map[string]string{"name": "test"})
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})

			It("should return 401 for DELETE /api/v1/events/:id without token", func() {
				w := helper.DoRequest(http.MethodDelete, "/api/v1/events/00000000-0000-0000-0000-000000000001", "", nil)
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("check-in endpoint", func() {
			It("should return 401 for POST /api/v1/events/:id/checkin without token", func() {
				w := helper.DoRequest(http.MethodPost, "/api/v1/events/00000000-0000-0000-0000-000000000001/checkin", "", nil)
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})

	When("registering duplicate users", func() {
		Context("with the same email address", func() {
			It("should return 409 Conflict on second registration", func() {
				helper.RegisterUser("dup@e2e-test.com", "Password123!", "First User", "organizer")

				// Second registration with same email
				w := helper.DoRequest(http.MethodPost, "/api/v1/auth/register", "", map[string]interface{}{
					"email":    "dup@e2e-test.com",
					"password": "Password123!",
					"name":     "Second User",
					"role":     "organizer",
				})
				Expect(w.Code).To(Equal(http.StatusConflict))
			})
		})
	})

	When("adding duplicate participants to an event", func() {
		Context("with the same email within the same event", func() {
			It("should return 409 Conflict on second registration", func() {
				auth := helper.RegisterUser("org@e2e-test.com", "Password123!", "Organizer", "organizer")
				event := helper.CreateEvent(auth.AccessToken, "Test Event")
				eventID := event.Id.String()

				helper.CreateParticipant(auth.AccessToken, eventID, "Participant A", "dup-p@e2e-test.com")

				// Same email, same event
				w := helper.DoRequest(http.MethodPost, "/api/v1/events/"+eventID+"/participants",
					auth.AccessToken, map[string]string{
						"name":  "Participant A Dup",
						"email": "dup-p@e2e-test.com",
					})
				Expect(w.Code).To(Equal(http.StatusConflict))
			})
		})
	})

	When("a non-owner tries to modify another organizer's event", func() {
		Context("with a different organizer's token", func() {
			It("should return 403 Forbidden", func() {
				owner := helper.RegisterUser("owner@e2e-test.com", "Password123!", "Owner", "organizer")
				other := helper.RegisterUser("other@e2e-test.com", "Password123!", "Other", "organizer")

				event := helper.CreateEvent(owner.AccessToken, "Owner's Event")
				eventID := event.Id.String()

				// Other organizer tries to delete owner's event
				w := helper.DoRequest(http.MethodDelete, "/api/v1/events/"+eventID, other.AccessToken, nil)
				Expect(w.Code).To(Equal(http.StatusForbidden))
			})
		})
	})

	When("using invalid credentials", func() {
		Context("with wrong password", func() {
			It("should return 401 on login", func() {
				helper.RegisterUser("creds@e2e-test.com", "Password123!", "Creds User", "organizer")

				w := helper.DoRequest(http.MethodPost, "/api/v1/auth/login", "", map[string]string{
					"email":    "creds@e2e-test.com",
					"password": "WrongPassword!",
				})
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})
})
