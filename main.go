package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	luks "github.com/anatol/luks.go"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	"github.com/kairos-io/kcrypt/pkg/bus"
	"github.com/mudler/go-pluggable"
	cp "github.com/otiai10/copy"
	"github.com/urfave/cli"
)

// TODO: Ask to discovery a pass to unlock. keep waiting until we get it and a timeout is exhausted with retrials (exp backoff)
func getPassword(b *block.Partition) (password string, err error) {
	bus.Reload()

	bus.Manager.Response(bus.EventDiscoveryPassword, func(p *pluggable.Plugin, r *pluggable.EventResponse) {
		password = r.Data
		if r.Errored() {
			err = fmt.Errorf("failed discovery: %s", r.Error)
		}
	})
	bus.Manager.Publish(bus.EventDiscoveryPassword, b)

	if password == "" {
		return password, fmt.Errorf("received empty password")
	}

	return
}

func luksUnlock(device, mapper, password string) error {
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

func unlockDisk(b *block.Partition) error {
	pass, err := getPassword(b)
	if err != nil {
		return fmt.Errorf("error retreiving password remotely: %w", err)
	}

	return luksUnlock(fmt.Sprintf("/dev/%s", b.Name), b.Name, pass)
}

func createLuks(dev, password, version string, cryptsetupArgs ...string) error {
	if version == "" {
		version = "luks2"
	}
	args := []string{"luksFormat", "--type", version, "--iter-time", "5", "-q", dev}
	args = append(args, cryptsetupArgs...)
	cmd := exec.Command("cryptsetup", args...)
	cmd.Stdin = strings.NewReader(password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func createDiskImage() (*os.File, error) {
	disk, err := ioutil.TempFile("", "luksv2.go.disk")
	if err != nil {
		return nil, err
	}

	if err := disk.Truncate(24 * 1024 * 1024); err != nil {
		return nil, err
	}

	return disk, err
}

// TODO: A crypt disk utility to call after install, that with discovery discoveries the password that should be used
// this function should delete COS_PERSISTENT. delete the partition and create a luks+type in place.

// Take a part label, and recreates it with LUKS. IT OVERWRITES DATA!
func luksify(label string) error {
	// blkid
	persistent, b, err := findPartition(label)
	if err != nil {
		return err
	}

	pass, err := getPassword(b)
	if err != nil {
		return err
	}

	persistent = fmt.Sprintf("/dev/%s", persistent)

	if err := createLuks(persistent, pass, "luks1"); err != nil {
		return err
	}

	if err := luksUnlock(persistent, b.Name, pass); err != nil {
		return err
	}

	out, err := sh(fmt.Sprintf("mkfs.ext4 %s -L %s", fmt.Sprintf("/dev/mapper/%s", b.Name), label))

	if err != nil {
		return fmt.Errorf("err: %w, out: %s", err, out)
	}

	out2, err := sh(fmt.Sprintf("cryptsetup close %s", b.Name))
	if err != nil {
		return fmt.Errorf("err: %w, out: %s", err, out2)
	}

	return nil
}

func findPartition(label string) (string, *block.Partition, error) {
	block, err := ghw.Block()
	if err == nil {
		for _, disk := range block.Disks {
			for _, p := range disk.Partitions {
				if p.Label == label {
					return p.Name, p, nil
				}

			}
		}
	} else {
		return "", nil, err
	}

	return "", nil, fmt.Errorf("not found")
}

func sh(c string) (string, error) {
	o, err := exec.Command("/bin/sh", "-c", c).CombinedOutput()
	return string(o), err
}

const (
	GZType   = "gz"
	XZType   = "xz"
	LZMAType = "lzma"
)

// TODO: replace with golang native code
func detect(archive string) (string, error) {
	out, err := sh(fmt.Sprintf("file %s", archive))
	if err != nil {
		return "", err
	}
	out = strings.ToLower(out)
	if strings.Contains(out, "xz") {
		return XZType, nil

	} else if strings.Contains(out, "lzma") {
		return LZMAType, nil

	} else if strings.Contains(out, "gz") {
		return GZType, nil

	}

	return "", fmt.Errorf("Unknown")
}

// TODO: replace with golang native code
func extractInitrd(initrd string, dst string) error {
	os.MkdirAll(dst, os.ModePerm)
	var out string
	var err error
	format, err := detect(initrd)
	if err != nil {
		return err
	}
	if format == XZType || format == LZMAType {
		out, err = sh(fmt.Sprintf("cd %s && xz -dc <  %s | cpio -idmv", dst, initrd))
	} else if format == GZType {
		out, err = sh(fmt.Sprintf("cd %s && zcat %s | cpio -idmv", dst, initrd))
	}
	fmt.Println(out)

	return err
}

func createInitrd(initrd string, src string, format string) error {
	fmt.Printf("Creating '%s' from '%s' as '%s'\n", initrd, src, format)

	if _, err := os.Stat(src); err != nil {
		return err
	}
	var err error
	var out string
	if format == XZType {
		out, err = sh(fmt.Sprintf("cd %s && find . 2>/dev/null | cpio -H newc --quiet --null -o -R root:root | xz -0 --check=crc32 > %s", src, initrd))
	} else if format == GZType {
		out, err = sh(fmt.Sprintf("cd %s && find . | cpio -H newc -o -R root:root | gzip -9 > %s", src, initrd))
	} else if format == LZMAType {
		out, err = sh(fmt.Sprintf("cd %s && find . 2>/dev/null | cpio -H newc -o -R root:root | xz -9 --format=lzma > %s", src, initrd))
	}
	fmt.Println(out)

	return err
}

// TODO: A inject initramfs command to add the discovery e.g. to use inside Dockerfiles

func injectInitrd(initrd string, file, dst string) error {

	fmt.Printf("Injecting '%s' as '%s' into '%s'\n", file, dst, initrd)
	format, err := detect(initrd)
	if err != nil {
		return err
	}
	tmp, err := ioutil.TempDir("", "kcrypt")
	if err != nil {
		return fmt.Errorf("cannot create tempdir, %s", err)
	}
	defer os.RemoveAll(tmp)

	fmt.Printf("Extracting '%s' in '%s' ...\n", initrd, tmp)
	if err := extractInitrd(initrd, tmp); err != nil {
		return fmt.Errorf("cannot extract initrd, %s", err)
	}

	d := filepath.Join(tmp, dst)
	fmt.Printf("Copying '%s' in '%s' ...\n", file, d)
	if err := cp.Copy(file, d); err != nil {
		return fmt.Errorf("cannot copy file, %s", err)
	}

	return createInitrd(initrd, tmp, format)
}

// TODO: a custom toolkit version, to build out initrd pre-built with this component
func unlockAll() error {
	bus.Manager.Initialize()

	block, err := ghw.Block()
	if err == nil {
		for _, disk := range block.Disks {
			for _, p := range disk.Partitions {
				if p.Type == "crypto_LUKS" {
					fmt.Printf("Unmounted Luks found at '%s' LABEL '%s' \n", p.Name, p.Label)
					err = multierror.Append(err, unlockDisk(p))
					if err != nil {
						fmt.Printf("Unlocking failed: '%s'\n", err.Error())
					}
					time.Sleep(10 * time.Second)
				}
			}
		}
	}
	return err
}

func main() {
	app := &cli.App{
		Name:        "keiros-kcrypt",
		Version:     "0.1",
		Author:      "Ettore Di Giacinto",
		Usage:       "keiros escrow key agent component",
		Description: ``,
		UsageText:   ``,
		Copyright:   "Ettore Di Giacinto",
		Commands: []cli.Command{
			{

				Name: "extract-initrd",
				Action: func(c *cli.Context) error {
					if c.NArg() != 2 {
						return fmt.Errorf("requires 3 args. initrd,, dst")
					}
					return extractInitrd(c.Args()[0], c.Args()[1])
				},
			},
			{

				Name:        "encrypt",
				Description: "Encrypts a partition",
				Action: func(c *cli.Context) error {
					if c.NArg() != 1 {
						return fmt.Errorf("requires 1 arg, the partition label")
					}
					return luksify(c.Args().First())
				},
			},
			{

				Name: "inject-initrd",
				Action: func(c *cli.Context) error {
					if c.NArg() != 3 {
						return fmt.Errorf("requires 3 args. initrd, srcfile, dst")
					}
					return injectInitrd(c.Args()[0], c.Args()[1], c.Args()[2])
				},
			},
			{
				Name:      "unlock-all",
				UsageText: "unlock-all",
				Usage:     "Try to unlock all LUKS partitions",
				Description: `
Typically run during initrd to unlock all the LUKS partitions found
		`,
				ArgsUsage: "kcrypt unlock-all",
				Flags: []cli.Flag{

					&cli.StringFlag{},
				},
				Action: func(c *cli.Context) error {
					return unlockAll()
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
