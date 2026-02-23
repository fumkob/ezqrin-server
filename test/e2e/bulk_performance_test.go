//go:build integration
// +build integration

// test/e2e/bulk_performance_test.go
package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/fumkob/ezqrin-server/test/fixtures"
	"github.com/fumkob/ezqrin-server/test/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bulk Operations Performance", func() {
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

	When("bulk importing participants", func() {
		Context("with 100 participants", func() {
			It("should complete within 10 seconds", func() {
				auth := helper.RegisterUser("bulk-org@e2e-test.com", "Password123!", "Bulk Org", "organizer")
				event := helper.CreateEvent(auth.AccessToken, "Bulk Import Event")
				eventID := event.Id.String()

				// Build bulk request with 100 participants
				type BulkParticipant struct {
					Name  string `json:"name"`
					Email string `json:"email"`
				}
				participants := make([]BulkParticipant, 100)
				for i := 0; i < 100; i++ {
					participants[i] = BulkParticipant{
						Name:  fmt.Sprintf("Participant %03d", i),
						Email: fmt.Sprintf("bulk-p%03d@e2e-test.com", i),
					}
				}
				reqBody := map[string]interface{}{"participants": participants}
				body, _ := json.Marshal(reqBody)

				req := httptest.NewRequest(http.MethodPost,
					"/api/v1/events/"+eventID+"/participants/bulk",
					bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
				w := httptest.NewRecorder()

				start := time.Now()
				env.Router.ServeHTTP(w, req)
				elapsed := time.Since(start)

				Expect(w.Code).To(Equal(http.StatusCreated),
					"Bulk import should succeed: %s", w.Body.String())
				Expect(elapsed).To(BeNumerically("<", 10*time.Second),
					"Bulk import of 100 participants should complete within 10 seconds")

				GinkgoWriter.Printf("Bulk import of 100 participants took %v\n", elapsed)
			})
		})

		Context("with 50 sequential individual registrations", func() {
			It("should complete within 30 seconds", func() {
				auth := helper.RegisterUser("seq-org@e2e-test.com", "Password123!", "Seq Org", "organizer")
				event := helper.CreateEvent(auth.AccessToken, "Sequential Import Event")
				eventID := event.Id.String()

				start := time.Now()
				for i := 0; i < 50; i++ {
					helper.CreateParticipant(auth.AccessToken, eventID,
						fmt.Sprintf("Seq Participant %03d", i),
						fmt.Sprintf("seq-p%03d@e2e-test.com", i),
					)
				}
				elapsed := time.Since(start)

				Expect(elapsed).To(BeNumerically("<", 30*time.Second),
					"50 sequential individual registrations should complete within 30 seconds")

				GinkgoWriter.Printf("50 sequential participant creations took %v (avg %v each)\n",
					elapsed, elapsed/50)
			})
		})
	})
})
