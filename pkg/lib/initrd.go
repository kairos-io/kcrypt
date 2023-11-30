package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cp "github.com/otiai10/copy"
)

const (
	GZType   = "gz"
	XZType   = "xz"
	LZMAType = "lzma"
)

func createInitrd(initrd string, src string, format string) error {
	fmt.Printf("Creating '%s' from '%s' as '%s'\n", initrd, src, format)

	if _, err := os.Stat(src); err != nil {
		return err
	}
	var err error
	var out string
	if format == XZType {
		out, err = SH(fmt.Sprintf("cd %s && find . 2>/dev/null | cpio -H newc --quiet --null -o -R root:root | xz -0 --check=crc32 > %s", src, initrd))
	} else if format == GZType {
		out, err = SH(fmt.Sprintf("cd %s && find . | cpio -H newc -o -R root:root | gzip -9 > %s", src, initrd))
	} else if format == LZMAType {
		out, err = SH(fmt.Sprintf("cd %s && find . 2>/dev/null | cpio -H newc -o -R root:root | xz -9 --format=lzma > %s", src, initrd))
	}
	fmt.Println(out)

	return err
}

func InjectInitrd(initrd string, file, dst string) error {
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
	if err := ExtractInitrd(initrd, tmp); err != nil {
		return fmt.Errorf("cannot extract initrd, %s", err)
	}

	d := filepath.Join(tmp, dst)
	fmt.Printf("Copying '%s' in '%s' ...\n", file, d)
	if err := cp.Copy(file, d); err != nil {
		return fmt.Errorf("cannot copy file, %s", err)
	}

	return createInitrd(initrd, tmp, format)
}

func ExtractInitrd(initrd string, dst string) error {
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
		out, err = SH(fmt.Sprintf("cd %s && xz -dc <  %s | cpio -idmv", dst, initrd))
	} else if format == GZType {
		out, err = SH(fmt.Sprintf("cd %s && zcat %s | cpio -idmv", dst, initrd))
	}
	fmt.Println(out)

	return err
}

func detect(archive string) (string, error) {
	out, err := SH(fmt.Sprintf("file %s", archive))
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
