package main

import (
	"fmt"
	"os"

	"github.com/kairos-io/kcrypt/pkg/lib"
	"github.com/urfave/cli"
)

var Version = "v0.0.0-dev"

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

				Name:        "encrypt",
				Description: "Encrypts a partition",
				Action: func(c *cli.Context) error {
					if c.NArg() != 1 {
						return fmt.Errorf("requires 1 arg, the partition label")
					}
					out, err := lib.Luksify(c.Args().First())
					if err != nil {
						return err
					}
					fmt.Println(out)
					return nil
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
			{

				Name: "extract-initrd",
				Action: func(c *cli.Context) error {
					if c.NArg() != 2 {
						return fmt.Errorf("requires 3 args. initrd,, dst")
					}
					return lib.ExtractInitrd(c.Args()[0], c.Args()[1])
				},
			},
			{

				Name: "inject-initrd",
				Action: func(c *cli.Context) error {
					if c.NArg() != 3 {
						return fmt.Errorf("requires 3 args. initrd, srcfile, dst")
					}
					return lib.InjectInitrd(c.Args()[0], c.Args()[1], c.Args()[2])
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
