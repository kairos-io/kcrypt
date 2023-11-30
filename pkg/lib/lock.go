package lib

import (
	"fmt"
	"github.com/anatol/luks.go"
	"github.com/gofrs/uuid"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	configpkg "github.com/kairos-io/kcrypt/pkg/config"
	"k8s.io/apimachinery/pkg/util/rand"
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

// Luksify Take a part label, and recreates it with LUKS. IT OVERWRITES DATA!
// On success, it returns a machine parseable string with the partition information
// (label:name:uuid) so that it can be stored by the caller for later use.
// This is because the label of the encrypted partition is not accessible unless
// the partition is decrypted first and the uuid changed after encryption so
// any stored information needs to be updated (by the caller).
func Luksify(label, version string, tpm bool) (string, error) {
	var pass string
	if version == "" {
		version = "luks1"
	}
	if version != "luks1" && version != "luks2" {
		return "", fmt.Errorf("version must be luks1 or luks2")
	}

	part, b, err := FindPartition(label)
	if err != nil {
		return "", err
	}

	if tpm {
		// On TPM locking we generate a random password that will only be used here then discarded.
		// only unlocking method will be PCR values
		pass = rand.String(32)
	} else {
		pass, err = GetPassword(b)
		if err != nil {
			return "", err
		}
	}

	part = fmt.Sprintf("/dev/%s", part)
	devMapper := fmt.Sprintf("/dev/mapper/%s", b.Name)
	partUUID := uuid.NewV5(uuid.NamespaceURL, label)

	extraArgs := []string{"--uuid", partUUID.String()}

	if err := CreateLuks(part, pass, version, extraArgs...); err != nil {
		return "", err
	}
	if tpm {
		// Enroll PCR values as an unlock method
		out, err := SH(fmt.Sprintf("systemd-cryptenroll --tpm2-device=auto --tpm2-pcrs=7+8+9 %s", part))
		if err != nil {
			return "", fmt.Errorf("err: %w, out: %s", err, out)
		}
	}

	if err := LuksUnlock(part, b.Name, pass); err != nil {
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

	err = luks.Lock(b.Name)
	if err != nil {
		return "", fmt.Errorf("err: %w", err)
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
