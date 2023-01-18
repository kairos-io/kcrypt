package config

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPartitionINfo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kcrypt config test suite")
}
