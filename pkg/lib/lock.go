package lib

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	configpkg "github.com/kairos-io/kcrypt/pkg/config"
	"os"
	"os/exec"
	"strings"
)

func CreateLuks(dev, password, version string, cryptsetupArgs ...string) error {
	if version == "" {
		version = "luks2"
	}
	args := []string{"luksFormat", "--type", version, "--iter-time", "5", "-q", dev}
	args = append(args, cryptsetupArgs...)
	cmd := exec.Command("cryptsetup", args...)
	cmd.Stdin = strings.NewReader(password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// Take a part label, and recreates it with LUKS. IT OVERWRITES DATA!
// On success, it returns a machine parseable string with the partition information
// (label:name:uuid) so that it can be stored by the caller for later use.
// This is because the label of the encrypted partition is not accessible unless
// the partition is decrypted first and the uuid changed after encryption so
// any stored information needs to be updated (by the caller).
func Luksify(label string) (string, error) {
	// blkid
	persistent, b, err := FindPartition(label)
	if err != nil {
		return "", err
	}

	pass, err := GetPassword(b)
	if err != nil {
		return "", err
	}

	persistent = fmt.Sprintf("/dev/%s", persistent)
	devMapper := fmt.Sprintf("/dev/mapper/%s", b.Name)
	partUUID := uuid.NewV5(uuid.NamespaceURL, label)

	if err := CreateLuks(persistent, pass, "luks1", []string{"--uuid", partUUID.String()}...); err != nil {
		return "", err
	}

	if err := LuksUnlock(persistent, b.Name, pass); err != nil {
		return "", err
	}

	if err := Waitdevice(devMapper, 10); err != nil {
		return "", err
	}

	cmd := fmt.Sprintf("mkfs.ext4 -L %s %s", label, devMapper)
	out, err := SH(cmd)
	if err != nil {
		return "", fmt.Errorf("err: %w, out: %s", err, out)
	}

	out2, err := SH(fmt.Sprintf("cryptsetup close %s", b.Name))
	if err != nil {
		return "", fmt.Errorf("err: %w, out: %s", err, out2)
	}

	return configpkg.PartitionToString(b), nil
}

func FindPartition(label string) (string, *block.Partition, error) {
	b, err := ghw.Block()
	if err == nil {
		for _, disk := range b.Disks {
			for _, p := range disk.Partitions {
				if p.FilesystemLabel == label {
					return p.Name, p, nil
				}

			}
		}
	} else {
		return "", nil, err
	}

	return "", nil, fmt.Errorf("not found")
}
