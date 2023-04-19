package lib

import (
	"fmt"
	"time"

	"github.com/anatol/luks.go"
	"github.com/hashicorp/go-multierror"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	"github.com/kairos-io/kcrypt/pkg/bus"
	configpkg "github.com/kairos-io/kcrypt/pkg/config"
	"github.com/mudler/go-pluggable"
)

// UnlockAll Unlocks all encrypted devices found in the system
func UnlockAll() error {
	bus.Manager.Initialize()

	config, err := configpkg.GetConfiguration(configpkg.ConfigScanDirs)
	if err != nil {
		fmt.Printf("Warning: Could not read kcrypt configuration '%s'\n", err.Error())
	}

	blk, err := ghw.Block()
	if err != nil {
		fmt.Printf("Warning: Error reading partitions '%s \n", err.Error())

		return nil
	}

	for _, disk := range blk.Disks {
		for _, p := range disk.Partitions {
			if p.Type == "crypto_LUKS" {
				p.Label = config.LookupLabelForUUID(p.UUID)
				fmt.Printf("Unmounted Luks found at '%s' LABEL '%s' \n", p.Name, p.Label)
				multiError := multierror.Append(err, UnlockDisk(p))
				if multiError.ErrorOrNil() != nil {
					fmt.Printf("Unlocking failed: '%s'\n", err.Error())
				}
				time.Sleep(10 * time.Second)
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

	return LuksUnlock(fmt.Sprintf("/dev/%s", b.Name), b.Name, pass)
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
