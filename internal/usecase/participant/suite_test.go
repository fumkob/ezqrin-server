package participant_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestParticipant(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Participant Usecase Suite")
}
