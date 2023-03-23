// package config contains all the logic around kcrypt config
// This config includes everything below `kcrypt:` in the kairos config yaml
package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/jaypipes/ghw/pkg/block"
	"github.com/kairos-io/kairos/pkg/config/collector"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v1"
)

// There are the directories under which we expect to find kairos configuration.
// When we are booted from an iso (during installation), configuration is expected
// under `/oem`. When we are booting an installed system (in initramfs phase),
// the path is `/sysroot/oem`.
var ConfigScanDirs = []string{"/oem", "/sysroot/oem"}

// This file is "hardcoded" to `/oem` because we only use this at install time
// in which case the config is in `/oem`.
var MappingsFile = "/oem/91-kcrypt-mappings.yaml"

type Config struct {
	Kcrypt struct {
		UUIDLabelMappings map[string]string `yaml:"uuid_label_mappings,omitempty"`
	}
}

func PartitionToString(p *block.Partition) string {
	return fmt.Sprintf("%s:%s:%s", p.Label, p.Name, p.UUID)
}

// Takes a partition info string (as returned by PartitionToString) and return
// the partition label and the UUID
func partitionDataFromString(partitionStr string) (string, string, error) {
	parts := strings.Split(partitionStr, ":")
	if len(parts) != 3 {
		return "", "", errors.New("partition string not valid")
	}

	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[2]), nil
}

func GetConfiguration(configDirs []string) (Config, error) {
	var result Config

	o := &collector.Options{}

	if err := o.Apply(collector.Directories(configDirs...), collector.NoLogs); err != nil {
		return result, err
	}

	c, err := collector.Scan(o)
	if err != nil {
		return result, err
	}
	configStr, err := c.String()
	if err != nil {
		return result, err
	}
	if err = yaml.Unmarshal([]byte(configStr), &result); err != nil {
		return result, err
	}

	return result, nil
}

// SetMapping updates the Config with partition information for
// one partition. This doesn't persist on the file. WriteMappings needs to
// be called after all mapping are in the Config (possibly with multiple calls
// to this function).
func (c *Config) SetMapping(partitionInfo string) error {
	label, uuid, err := partitionDataFromString(partitionInfo)
	if err != nil {
		return err
	}
	// Initialize map
	if c.Kcrypt.UUIDLabelMappings == nil {
		c.Kcrypt.UUIDLabelMappings = map[string]string{}
	}
	c.Kcrypt.UUIDLabelMappings[label] = uuid

	return nil
}

// WriteMappings will create or replace the MappingsFile
// It's called by kairos agent, at installation time, after the partitions
// have been created (and we have the UUIDs available).
func (c *Config) WriteMappings(fileName string) error {
	data, err := yaml.Marshal(&c)
	if err != nil {
		return errors.Wrap(err, "marshalling the kcrypt configuration to yaml")
	}

	data = append([]byte(collector.DefaultHeader+"\n"), data...)

	err = ioutil.WriteFile(fileName, data, 0744)
	if err != nil {
		return errors.Wrap(err, "writing the kcrypt configuration file")
	}

	return nil
}

func (c Config) LookupUUIDForLabel(l string) string {
	return c.Kcrypt.UUIDLabelMappings[l]
}

func (c Config) LookupLabelForUUID(uuid string) string {
	for k, v := range c.Kcrypt.UUIDLabelMappings {
		if v == uuid {
			return k
		}
	}

	return ""
}
