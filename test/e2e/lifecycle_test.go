//go:build integration
// +build integration

// test/e2e/lifecycle_test.go
package e2e_test

import (
	"net/http"

	"github.com/fumkob/ezqrin-server/test/fixtures"
	"github.com/fumkob/ezqrin-server/test/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Full Event Lifecycle", func() {
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

	When("executing the full event management lifecycle", func() {
		Context("with organizer and multiple participants", func() {
			It("should successfully complete the entire flow", func() {
				By("Step 1: Register an organizer")
				organizerAuth := helper.RegisterUser(
					"organizer@lifecycle-test.com",
					"Password123!",
					"Test Organizer",
					"organizer",
				)
				Expect(organizerAuth.AccessToken).NotTo(BeEmpty())

				By("Step 2: Register a staff user")
				staffAuth := helper.RegisterUser(
					"staff@lifecycle-test.com",
					"Password123!",
					"Test Staff",
					"staff",
				)
				Expect(staffAuth.AccessToken).NotTo(BeEmpty())

				By("Step 3: Create an event")
				event := helper.CreateEvent(organizerAuth.AccessToken, "Annual Conference 2026")
				Expect(event.Name).To(Equal("Annual Conference 2026"))
				eventID := event.Id.String()

				By("Step 4: Add 4 participants to the event")
				p1 := helper.CreateParticipant(organizerAuth.AccessToken, eventID,
					"Alice Smith", "alice@lifecycle-test.com")
				p2 := helper.CreateParticipant(organizerAuth.AccessToken, eventID,
					"Bob Johnson", "bob@lifecycle-test.com")
				p3 := helper.CreateParticipant(organizerAuth.AccessToken, eventID,
					"Carol Williams", "carol@lifecycle-test.com")
				p4 := helper.CreateParticipantWithEmployeeID(organizerAuth.AccessToken, eventID,
					"Dave Employee", "dave@lifecycle-test.com", "EMP001")

				Expect(p1.QrCode).NotTo(BeNil(), "QR code should be auto-generated")
				Expect(p2.QrCode).NotTo(BeNil())
				Expect(p3.QrCode).NotTo(BeNil())
				Expect(*p1.QrCode).NotTo(Equal(*p2.QrCode), "QR codes must be globally unique")
				Expect(p4.EmployeeId).NotTo(BeNil())
				Expect(*p4.EmployeeId).To(Equal("EMP001"))

				By("Step 5: Check in participant 1 by QR code")
				checkin1, status1 := helper.CheckInByQR(
					organizerAuth.AccessToken, eventID, *p1.QrCode,
				)
				Expect(status1).To(Equal(http.StatusOK))
				Expect(checkin1.ParticipantId.String()).To(Equal(p1.Id.String()))
				Expect(string(checkin1.CheckinMethod)).To(Equal("qrcode"))

				By("Step 6: Check in participant 2 by manual (participant ID, organizer only)")
				// Manual check-in requires the organizer or admin: staff role is not permitted
				checkin2, status2 := helper.CheckInByManual(
					organizerAuth.AccessToken, eventID, p2.Id.String(),
				)
				Expect(status2).To(Equal(http.StatusOK))
				Expect(checkin2.ParticipantId.String()).To(Equal(p2.Id.String()))
				Expect(string(checkin2.CheckinMethod)).To(Equal("manual"))

				By("Step 7: Check in participant 4 by employee ID")
				checkin4, status4 := helper.CheckInByEmployeeID(
					organizerAuth.AccessToken, eventID, "EMP001",
				)
				Expect(status4).To(Equal(http.StatusOK))
				Expect(checkin4.ParticipantId.String()).To(Equal(p4.Id.String()))
				Expect(string(checkin4.CheckinMethod)).To(Equal("manual"))

				By("Step 8: Verify event statistics reflect 3 check-ins out of 4 participants")
				stats, statsStatus := helper.GetEventStats(organizerAuth.AccessToken, eventID)
				Expect(statsStatus).To(Equal(http.StatusOK))
				Expect(stats.TotalParticipants).To(Equal(4))
				Expect(stats.CheckedInParticipants).To(Equal(3))

				By("Step 9: Staff user can perform QR check-in on participant 3")
				// Staff role is allowed for QR check-in (only manual requires organizer/admin)
				checkin3, status3 := helper.CheckInByQR(
					staffAuth.AccessToken, eventID, *p3.QrCode,
				)
				Expect(status3).To(Equal(http.StatusOK))
				Expect(checkin3.ParticipantId.String()).To(Equal(p3.Id.String()))

				By("Step 10: Duplicate check-in should be rejected")
				_, dupStatus := helper.CheckInByQR(
					organizerAuth.AccessToken, eventID, *p1.QrCode,
				)
				Expect(dupStatus).To(Equal(http.StatusConflict))
			})
		})
	})
})

var _ = Describe("Authentication Flow", func() {
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

	When("executing the full authentication lifecycle", func() {
		Context("with valid credentials", func() {
			It("should complete register → login → use token → refresh → logout", func() {
				email := "auth-flow@e2e-test.com"
				password := "Password123!"

				By("Register a new user")
				registerResp := helper.RegisterUser(email, password, "Auth Flow User", "organizer")
				Expect(registerResp.AccessToken).NotTo(BeEmpty())
				Expect(registerResp.RefreshToken).NotTo(BeEmpty())

				By("Use the access token to access a protected resource")
				w := helper.DoRequest(http.MethodGet, "/api/v1/events", registerResp.AccessToken, nil)
				Expect(w.Code).To(Equal(http.StatusOK))

				By("Refresh the access token")
				refreshBody := map[string]string{"refresh_token": registerResp.RefreshToken}
				w = helper.DoRequest(http.MethodPost, "/api/v1/auth/refresh", "", refreshBody)
				Expect(w.Code).To(Equal(http.StatusOK))

				By("Logout")
				w = helper.DoRequest(http.MethodPost, "/api/v1/auth/logout",
					registerResp.AccessToken, nil)
				Expect(w.Code).To(Equal(http.StatusOK))

				By("Verify blacklisted token is rejected (re-login to check)")
				// After logout, accessing with the old token should be rejected
				// Note: Some endpoints may be publicly accessible - use a protected one
				w = helper.DoRequest(http.MethodPost, "/api/v1/auth/logout",
					registerResp.AccessToken, nil)
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})
})
