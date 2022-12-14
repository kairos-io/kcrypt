package partition_info_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

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
		When("the files exists already", func() {
			It("returns 'true' and a PartitionInfo object", func() {
				result, existed, err := pi.NewPartitionInfoFromFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(existed).To(BeTrue())
			})
		})

		When("a file doesn't exist", func() {
			var fileName string
			BeforeEach(func() {
				fileName = path.Join(
					os.TempDir(),
					fmt.Sprintf("partition-info-%d.yaml", time.Now().UnixNano()))
			})

			When("there is some error other than the file doesn't exist", func() {
				It("returns 'false' and the error", func() {
					// We are trying to write to a path that doesn't exist (not the file, the path).
					// https://stackoverflow.com/a/66808328
					fileName = "\000x"
					_, _, err := pi.NewPartitionInfoFromFile(fileName)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("stat \000x: invalid argument"))
				})
			})

			It("creates the file and returns 'false' and a non nil mapping", func() {
				result, existed, err := pi.NewPartitionInfoFromFile(fileName)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(existed).To(BeFalse())
				_, err = os.Stat(fileName)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.IsMappingNil()).To(BeFalse())
			})
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

			partitionInfo, _, err = pi.NewPartitionInfoFromFile(file.Name())
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
			partitionInfo, _, err = pi.NewPartitionInfoFromFile(file)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the correct UUID", func() {
			uuid := partitionInfo.LookupUUIDForLabel("COS_PERSISTENT")
			Expect(uuid).To(Equal("some_uuid_1"))
		})

		It("returns an empty UUID when the label is not found", func() {
			uuid := partitionInfo.LookupUUIDForLabel("DOESNT_EXIST")
			Expect(uuid).To(Equal(""))
		})
	})

	Describe("LookupLabelForUUID", func() {
		var file string
		var partitionInfo *pi.PartitionInfo
		var err error

		BeforeEach(func() {
			file = "../../tests/assets/partition_info.yaml"
			partitionInfo, _, err = pi.NewPartitionInfoFromFile(file)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the correct label", func() {
			uuid := partitionInfo.LookupLabelForUUID("some_uuid_1")
			Expect(uuid).To(Equal("COS_PERSISTENT"))
		})

		It("returns an empty label when UUID doesn't exist", func() {
			uuid := partitionInfo.LookupLabelForUUID("doesnt_exist")
			Expect(uuid).To(Equal(""))
		})
	})
})
