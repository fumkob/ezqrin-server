package checkin_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCheckin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Checkin Suite")
}
