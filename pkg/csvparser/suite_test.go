package csvparser_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCSVParser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CSVParser Suite")
}
