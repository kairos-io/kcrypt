package partition_info

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPartitionINfo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PartitionInfo file parser test suite")
}
