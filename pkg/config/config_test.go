package config_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configpkg "github.com/kairos-io/kcrypt/pkg/config"
)

var _ = Describe("Config", func() {
	var tmpDir string
	var err error

	BeforeEach(func() {
		tmpDir, err = os.MkdirTemp("", "kcrypt-configuration-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("GetConfiguration", func() {
		When("the no relevant block exists", func() {
			It("returns empty Config", func() {
				c, err := configpkg.GetConfiguration([]string{tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(c.Kcrypt.UUIDLabelMappings).To(BeEmpty())
			})
		})

		When("a kcrypt block exists", func() {
			var tmpFile *os.File

			BeforeEach(func() {
				tmpFile, err = os.CreateTemp(tmpDir, "config-*.yaml")
				Expect(err).ToNot(HaveOccurred())
				data := []byte(`#cloud-config
kcrypt:
  uuid_label_mappings:
    COS_PERSISTENT: some_uuid_here
`)
				err := os.WriteFile(tmpFile.Name(), data, 0744)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the Config", func() {
				c, err := configpkg.GetConfiguration([]string{tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(c.Kcrypt.UUIDLabelMappings["COS_PERSISTENT"]).To(Equal("some_uuid_here"))
			})
		})

		When("multiple kcrypt block exist", func() {
			var tmpFile1, tmpFile2 *os.File

			BeforeEach(func() {
				tmpFile1, err = os.CreateTemp(tmpDir, "config-*.yaml")
				Expect(err).ToNot(HaveOccurred())
				data := []byte(`#cloud-config
kcrypt:
  challenger_server: http://test.org:8082
`)
				err := os.WriteFile(tmpFile1.Name(), data, 0744)
				Expect(err).ToNot(HaveOccurred())

				tmpFile2, err = os.CreateTemp(tmpDir, "config-*.yaml")
				Expect(err).ToNot(HaveOccurred())
				data = []byte(`#cloud-config
kcrypt:
  uuid_label_mappings:
    COS_PERSISTENT: some_uuid_here
`)
				err = os.WriteFile(tmpFile2.Name(), data, 0744)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the merged Config", func() {
				c, err := configpkg.GetConfiguration([]string{tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(c.Kcrypt.UUIDLabelMappings["COS_PERSISTENT"]).To(Equal("some_uuid_here"))
			})
		})
	})

	Describe("SetMapping", func() {
		var c configpkg.Config

		BeforeEach(func() {
			c, err = configpkg.GetConfiguration([]string{tmpDir})
			Expect(err).ToNot(HaveOccurred())
		})

		It("adds partition information when empty and appends when not", func() {
			Expect(c.Kcrypt.UUIDLabelMappings).To(BeNil())
			err := c.SetMapping("some_label:some_name:some_uuid")
			Expect(err).ToNot(HaveOccurred())
			Expect(c.Kcrypt.UUIDLabelMappings["some_label"]).To(Equal("some_uuid"))

			err = c.SetMapping("some_other_label:some_name:some_other_uuid")
			Expect(err).ToNot(HaveOccurred())
			Expect(c.Kcrypt.UUIDLabelMappings["some_label"]).To(Equal("some_uuid"))
			Expect(c.Kcrypt.UUIDLabelMappings["some_other_label"]).To(Equal("some_other_uuid"))
		})
	})

	Describe("WriteMappings", func() {
		var tmpFile *os.File
		var c configpkg.Config

		When("mappings config file already exists", func() {
			BeforeEach(func() {
				tmpFile, err = os.CreateTemp(tmpDir, "config-*.yaml")
				Expect(err).ToNot(HaveOccurred())
				data := []byte(`kcrypt:
  uuid_label_mappings:
    COS_PERSISTENT: some_uuid_here
`)
				err := os.WriteFile(tmpFile.Name(), data, 0744)
				Expect(err).ToNot(HaveOccurred())
			})

			It("replaces the file contents", func() {
				c.SetMapping("COS_PERSISTENT:the_new_name:the_new_uuid")
				c.WriteMappings(tmpFile.Name())

				newConfig, err := configpkg.GetConfiguration([]string{tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(newConfig.Kcrypt.UUIDLabelMappings["COS_PERSISTENT"]).To(Equal("the_new_uuid"))
			})
		})

		When("a mappings configuration file doesn't exist", func() {
			BeforeEach(func() {
				tmpFile, err = os.CreateTemp(tmpDir, "config-*.yaml")
				Expect(err).ToNot(HaveOccurred())
				// We will reuse the same name but we make sure the file doesn't exist.
				os.RemoveAll(tmpFile.Name())
			})

			It("creates the file with the given mappings", func() {
				c.SetMapping("COS_PERSISTENT:the_new_name:the_new_uuid")
				err := c.WriteMappings(tmpFile.Name())
				Expect(err).ToNot(HaveOccurred())

				newConfig, err := configpkg.GetConfiguration([]string{tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(newConfig.Kcrypt.UUIDLabelMappings["COS_PERSISTENT"]).To(Equal("the_new_uuid"))
			})
		})
	})

	Describe("LookupUUIDForLabel", func() {
		var tmpFile *os.File
		var c configpkg.Config

		BeforeEach(func() {
			tmpFile, err = os.CreateTemp(tmpDir, "config-*.yaml")
			Expect(err).ToNot(HaveOccurred())
			// Should trim the whitespace
			c.SetMapping("COS_PERSISTENT:the_new_name:some_uuid_1\n")
			c.WriteMappings(tmpFile.Name())
		})

		It("returns the correct UUID", func() {
			uuid := c.LookupUUIDForLabel("COS_PERSISTENT")
			Expect(uuid).To(Equal("some_uuid_1"))
		})

		It("returns an empty UUID when the label is not found", func() {
			uuid := c.LookupUUIDForLabel("DOESNT_EXIST")
			Expect(uuid).To(Equal(""))
		})

		It("returns an empty UUID when the UUIDLabelMappings is nil", func() {
			c.Kcrypt.UUIDLabelMappings = nil
			uuid := c.LookupUUIDForLabel("COS_PERSISTENT")
			Expect(uuid).To(Equal(""))
		})
	})

	Describe("LookupLabelForUUID", func() {
		var tmpFile *os.File
		var c configpkg.Config

		BeforeEach(func() {
			tmpFile, err = os.CreateTemp(tmpDir, "config-*.yaml")
			Expect(err).ToNot(HaveOccurred())
			c.SetMapping("COS_PERSISTENT:the_new_name:some_uuid_1")
			c.WriteMappings(tmpFile.Name())
		})

		It("returns the correct label", func() {
			uuid := c.LookupLabelForUUID("some_uuid_1")
			Expect(uuid).To(Equal("COS_PERSISTENT"))
		})

		It("returns an empty label when UUID doesn't exist", func() {
			uuid := c.LookupLabelForUUID("doesnt_exist")
			Expect(uuid).To(Equal(""))
		})
	})
})
