package partition_info

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/jaypipes/ghw/pkg/block"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const DefaultPartitionInfoFile = "/oem/partition_info.yaml"

// PartitionInfo maps a partition label to a partition UUID.
// It's used in order to be able to ask the kcrypt-challenger for the passphrase
// using the partition label, even when the label is not accessible (e.g. before
// decrypting the partition). The UUID can be used to lookup the partition label
// and make the request.
type PartitionInfo struct {
	file    string
	mapping map[string]string
}

func NewPartitionInfoFromFile(file string) (*PartitionInfo, error) {
	mapping, err := ParsePartitionInfoFile(file)
	if err != nil {
		return nil, err
	}

	return &PartitionInfo{
		file:    file,
		mapping: mapping,
	}, nil
}

func (pi PartitionInfo) LookupUUIDForLabel(l string) string {
	return pi.mapping[l]
}

func (pi PartitionInfo) LookupLabelForUUID(uuid string) string {
	for k, v := range pi.mapping {
		if v == uuid {
			return k
		}
	}

	return ""
}

// UpdatePartitionLabelMapping takes partition information as a string argument
// the the form: `label:name:uuid` (that's what the `kcrypt encrypt` command returns
// on success. This function stores it in the PartitionInfoFile yaml file for
// later use.
func (pi PartitionInfo) UpdateMapping(partitionData string) error {
	label, uuid := PartitionDataFromString(partitionData)
	pi.mapping[label] = uuid

	return pi.save()
}

func (pi PartitionInfo) save() error {
	data, err := yaml.Marshal(&pi.mapping)
	if err != nil {
		return errors.Wrap(err, "marshalling the new partition info to yaml")
	}
	err = ioutil.WriteFile(pi.file, data, 0)
	if err != nil {
		return errors.Wrap(err, "writing back the partition info file")
	}
	return nil
}

func PartitionToString(p *block.Partition) string {
	return fmt.Sprintf("%s:%s:%s", p.Label, p.Name, p.UUID)
}

// Takes a partition info string (as returned by PartitionToString) and return
// the partition label and the UUID
func PartitionDataFromString(partitionStr string) (string, string) {
	parts := strings.Split(partitionStr, ":")

	return parts[0], parts[2]
}

func ParsePartitionInfoFile(file string) (map[string]string, error) {
	var result map[string]string

	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		return result, errors.Wrap(err, "reading the partition info file")
	}

	err = yaml.Unmarshal(yamlFile, &result)
	if err != nil {
		return result, errors.Wrap(err, "unmarshalling partition info file")
	}

	return result, nil
}
