package partition_info_test

import (
	"io/ioutil"
	"os"

	"github.com/jaypipes/ghw/pkg/block"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pi "github.com/kairos-io/kcrypt/pkg/partition_info"
)

var _ = Describe("Partition Info file parsing", func() {
	Describe("NewPartitionInfoFromFile", func() {
		var file string

		BeforeEach(func() {
			file = "../../tests/assets/partition_info.yaml"
		})

		It("returns a PartitionInfo", func() {
			result, err := pi.NewPartitionInfoFromFile(file)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
		})
	})

	Describe("ParsePartitionInfoFile", func() {
		var file string

		BeforeEach(func() {
			file = "../../tests/assets/partition_info.yaml"
		})

		It("parses the file correctly", func() {
			info, err := pi.ParsePartitionInfoFile(file)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(info)).To(Equal(2))
			Expect(info["COS_PERSISTENT"]).To(Equal("some_uuid_1"))
			Expect(info["COS_OTHER"]).To(Equal("some_uuid_2"))
		})
	})

	Describe("PartitionToString", func() {
		var partition *block.Partition

		BeforeEach(func() {
			partition = &block.Partition{
				Disk:       nil,
				Name:       "sda1",
				Label:      "COS_PERSISTENT",
				MountPoint: "/mnt/sda1",
				UUID:       "some_uuid_here",
			}
		})

		It("returns a string representation of the partition data", func() {
			Expect(pi.PartitionToString(partition)).To(Equal("COS_PERSISTENT:sda1:some_uuid_here"))
		})
	})

	Describe("PartitionDataFromString", func() {
		var partitionData string

		BeforeEach(func() {
			partitionData = "THE_LABEL:the_name:the_uuid"
		})

		It("returns the label and the uuid", func() {
			label, uuid := pi.PartitionDataFromString(partitionData)
			Expect(label).To(Equal("THE_LABEL"))
			Expect(uuid).To(Equal("the_uuid"))
		})
	})

	Describe("UpdateMapping", func() {
		var file *os.File
		var err error
		var partitionInfo *pi.PartitionInfo

		BeforeEach(func() {
			file, err = ioutil.TempFile("", "partition-info.*.yaml")
			Expect(err).ToNot(HaveOccurred())

			_, err = file.Write([]byte("TO_KEEP: old_uuid_1"))
			Expect(err).ToNot(HaveOccurred())

			partitionInfo, err = pi.NewPartitionInfoFromFile(file.Name())
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.Remove(file.Name())
		})

		It("Updates the file correctly from a `kcrypt encrypt` return value", func() {
			partitionData := "TO_BE_ADDED:some_name:new_uuid"

			err = partitionInfo.UpdateMapping(partitionData)
			Expect(err).ToNot(HaveOccurred())

			dat, err := os.ReadFile(file.Name())
			Expect(err).ToNot(HaveOccurred())

			expectedContent := `TO_BE_ADDED: new_uuid
TO_KEEP: old_uuid_1
`
			Expect(string(dat)).To(Equal(expectedContent))
		})
	})

	Describe("LookupUUIDForLabel", func() {
		var file string
		var partitionInfo *pi.PartitionInfo
		var err error

		BeforeEach(func() {
			file = "../../tests/assets/partition_info.yaml"
			partitionInfo, err = pi.NewPartitionInfoFromFile(file)
			Expect(err).ToNot(HaveOccurred())
		})

		It("parses the file correctly", func() {
			uuid := partitionInfo.LookupUUIDForLabel("COS_PERSISTENT")
			Expect(uuid).To(Equal("some_uuid_1"))
		})
	})
})
