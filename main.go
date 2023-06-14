package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	configpkg "github.com/kairos-io/kcrypt/pkg/config"
	"github.com/kairos-io/kcrypt/pkg/lib"
	cp "github.com/otiai10/copy"
	"github.com/urfave/cli"
)

var Version = "v0.0.0-dev"

func waitdevice(device string, attempts int) error {
	for tries := 0; tries < attempts; tries++ {
		_, err := sh("udevadm settle")
		if err != nil {
			return err
		}
		_, err = os.Lstat(device)
		if !os.IsNotExist(err) {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("no device found")
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
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// TODO: A crypt disk utility to call after install, that with discovery discoveries the password that should be used
// this function should delete COS_PERSISTENT. delete the partition and create a luks+type in place.

// Take a part label, and recreates it with LUKS. IT OVERWRITES DATA!
// On success, it returns a machine parseable string with the partition information
// (label:name:uuid) so that it can be stored by the caller for later use.
// This is because the label of the encrypted partition is not accessible unless
// the partition is decrypted first and the uuid changed after encryption so
// any stored information needs to be updated (by the caller).
func luksify(label string) (string, error) {
	// blkid
	persistent, b, err := findPartition(label)
	if err != nil {
		return "", err
	}

	pass, err := lib.GetPassword(b)
	if err != nil {
		return "", err
	}

	persistent = fmt.Sprintf("/dev/%s", persistent)
	devMapper := fmt.Sprintf("/dev/mapper/%s", b.Name)
	partUUID := uuid.NewV5(uuid.NamespaceURL, label)

	if err := createLuks(persistent, pass, "luks1", []string{"--uuid", partUUID.String()}...); err != nil {
		return "", err
	}

	if err := lib.LuksUnlock(persistent, b.Name, pass); err != nil {
		return "", err
	}

	if err := waitdevice(devMapper, 10); err != nil {
		return "", err
	}

	cmd := fmt.Sprintf("mkfs.ext4 -L %s %s", label, devMapper)
	out, err := sh(cmd)
	if err != nil {
		return "", fmt.Errorf("err: %w, out: %s", err, out)
	}

	out2, err := sh(fmt.Sprintf("cryptsetup close %s", b.Name))
	if err != nil {
		return "", fmt.Errorf("err: %w, out: %s", err, out2)
	}

	return configpkg.PartitionToString(b), nil
}

func findPartition(label string) (string, *block.Partition, error) {
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
	var out string
	var err error
	err = os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return err
	}

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
	tmp, err := os.MkdirTemp("", "kcrypt")
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

func main() {
	app := &cli.App{
		Name:        "kairos-kcrypt",
		Version:     Version,
		Author:      "Ettore Di Giacinto",
		Usage:       "kairos escrow key agent component",
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
					out, err := luksify(c.Args().First())
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
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
					return lib.UnlockAll()
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
