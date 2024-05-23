package lib

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	configpkg "github.com/kairos-io/kcrypt/pkg/config"
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

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func getRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
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

	// Make sure ghw will see all partitions correctly
	out, err := SH("udevadm trigger --settle -v --type=all")
	if err != nil {
		return "", fmt.Errorf("udevadm trigger failed: %w, out: %s", err, out)
	}
	SH("sync") //nolint:errcheck

	part, b, err := FindPartition(label)
	if err != nil {
		return "", err
	}

	if tpm {
		// On TPM locking we generate a random password that will only be used here then discarded.
		// only unlocking method will be PCR values
		pass = getRandomString(32)
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
		// Enroll PCR policy as a keyslot
		// We pass the current signature of the booted system to confirm that we would be able to unlock with the current booted system
		// That checks the policy against the signatures and fails if a UKI with those signatures wont be able to unlock the device
		// Files are generated by systemd automatically and are extracted from the UKI binary directly
		// public pem cert -> .pcrpkey section fo the elf file
		// signatures -> .pcrsig section of the elf file
		args := []string{"--tpm2-public-key=/run/systemd/tpm2-pcr-public-key.pem", "--tpm2-signature=/run/systemd/tpm2-pcr-signature.json", "--tpm2-device=auto", part}
		cmd := exec.Command("systemd-cryptenroll", args...)
		cmd.Env = append(cmd.Env, fmt.Sprintf("PASSWORD=%s", pass)) // cannot pass it via stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return "", err
		}
	}

	if err := LuksUnlock(part, b.Name, pass); err != nil {
		return "", fmt.Errorf("unlock err: %w", err)
	}

	if err := Waitdevice(devMapper, 10); err != nil {
		return "", fmt.Errorf("waitdevice err: %w", err)
	}

	cmd := fmt.Sprintf("mkfs.ext4 -L %s %s", label, devMapper)
	out, err = SH(cmd)
	if err != nil {
		return "", fmt.Errorf("mkfs err: %w, out: %s", err, out)
	}

	out, err = SH(fmt.Sprintf("cryptsetup close %s", b.Name))
	if err != nil {
		return "", fmt.Errorf("lock err: %w, out: %s", err, out)
	}

	if tpm {
		// Delete password slot from luks device
		out, err := SH(fmt.Sprintf("systemd-cryptenroll --wipe-slot=password %s", part))
		if err != nil {
			return "", fmt.Errorf("err: %w, out: %s", err, out)
		}
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
