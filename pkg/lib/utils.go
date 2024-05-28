package lib

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func SH(c string) (string, error) {
	o, err := exec.Command("/bin/sh", "-c", c).CombinedOutput()
	return string(o), err
}

func Waitdevice(device string, attempts int) error {
	for tries := 0; tries < attempts; tries++ {
		_, err := SH("udevadm settle")
		if err != nil {
			return err
		}
		_, err = os.Lstat(device)
		if !os.IsNotExist(err) {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("no device found %s", device)
}
