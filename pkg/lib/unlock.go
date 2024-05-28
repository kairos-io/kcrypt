package lib

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/anatol/luks.go"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	"github.com/kairos-io/kairos-sdk/utils"
	"github.com/kairos-io/kcrypt/pkg/bus"
	configpkg "github.com/kairos-io/kcrypt/pkg/config"
	"github.com/mudler/go-pluggable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// UnlockAll Unlocks all encrypted devices found in the system
func UnlockAll(tpm bool) error {
	logger := log.Logger

	return UnlockAllWithLogger(tpm, logger)
}

func UnlockAllWithLogger(tpm bool, logger zerolog.Logger) error {
	bus.Manager.Initialize()

	config, err := configpkg.GetConfiguration(configpkg.ConfigScanDirs)
	if err != nil {
		logger.Info().Msgf("Warning: Could not read kcrypt configuration '%s'\n", err.Error())
	}

	blk, err := ghw.Block()
	if err != nil {
		logger.Warn().Msgf("Warning: Error reading partitions '%s \n", err.Error())

		return nil
	}

	// Some versions of udevadm don't support --settle (e.g. alpine)
	// and older versions don't have --type=all. Try the simpler version then.
	logger.Info().Msgf("triggering udev to populate disk info")
	_, err = utils.SH("udevadm trigger --settle --type=all || udevadm trigger")
	if err != nil {
		return err
	}

	for _, disk := range blk.Disks {
		for _, p := range disk.Partitions {
			if p.Type == "crypto_LUKS" {
				// Get the luks UUID directly from cryptsetup
				volumeUUID, err := utils.SH(fmt.Sprintf("cryptsetup luksUUID %s", filepath.Join("/dev", p.Name)))
				logger.Info().Msgf("Got luks UUID %s for partition %s\n", volumeUUID, p.Name)
				if err != nil {
					return err
				}
				volumeUUID = strings.TrimSpace(volumeUUID)
				if volumeUUID == "" {
					logger.Warn().Msgf("No uuid for %s, skipping\n", p.Name)
					continue
				}
				// Check if device is already mounted
				// We mount it under /dev/mapper/DEVICE, so It's pretty easy to check
				if !utils.Exists(filepath.Join("/dev", "mapper", p.Name)) {
					logger.Info().Msgf("Unmounted Luks found at '%s' \n", filepath.Join("/dev", p.Name))
					if tpm {
						out, err := utils.SH(fmt.Sprintf("/usr/lib/systemd/systemd-cryptsetup attach %s %s - tpm2-device=auto", p.Name, filepath.Join("/dev", p.Name)))
						if err != nil {
							logger.Warn().Msgf("Unlocking failed: '%s'\n", err.Error())
							logger.Warn().Msgf("Unlocking failed, command output: '%s'\n", out)
						}
					} else {
						p.FilesystemLabel, err = config.GetLabelForUUID(volumeUUID)
						if err != nil {
							return err
						}
						err = UnlockDisk(p)
						if err != nil {
							logger.Warn().Msgf("Unlocking failed: '%s'\n", err.Error())
						}
					}
				} else {
					logger.Info().Msgf("Device %s seems to be mounted at %s, skipping\n", filepath.Join("/dev", p.Name), filepath.Join("/dev", "mapper", p.Name))
				}

			}
		}
	}
	return nil
}

// UnlockDisk unlocks a single block.Partition
func UnlockDisk(b *block.Partition) error {
	pass, err := GetPassword(b)
	if err != nil {
		return fmt.Errorf("error retreiving password remotely: %w", err)
	}

	return LuksUnlock(filepath.Join("/dev", b.Name), b.Name, pass)
}

// GetPassword gets the password for a block.Partition
// TODO: Ask to discovery a pass to unlock. keep waiting until we get it and a timeout is exhausted with retrials (exp backoff)
func GetPassword(b *block.Partition) (password string, err error) {
	bus.Reload()

	bus.Manager.Response(bus.EventDiscoveryPassword, func(p *pluggable.Plugin, r *pluggable.EventResponse) {
		password = r.Data
		if r.Errored() {
			err = fmt.Errorf("failed discovery: %s", r.Error)
		}
	})
	_, err = bus.Manager.Publish(bus.EventDiscoveryPassword, b)
	if err != nil {
		return password, err
	}

	if password == "" {
		return password, fmt.Errorf("received empty password")
	}

	return
}

func LuksUnlock(device, mapper, password string) error {
	dev, err := luks.Open(device)
	if err != nil {
		// handle error
		return err
	}
	defer dev.Close()

	err = dev.Unlock(0, []byte(password), mapper)
	if err != nil {
		return err
	}
	return nil
}
