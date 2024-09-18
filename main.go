package main

import (
	"fmt"
	"os"

	"github.com/kairos-io/kairos-sdk/types"
	"github.com/kairos-io/kcrypt/pkg/lib"
	"github.com/urfave/cli/v2"
)

var Version = "v0.0.0-dev"

func main() {
	app := &cli.App{
		Name:        "kairos-kcrypt",
		Version:     Version,
		Authors:     []*cli.Author{&cli.Author{Name: "Ettore Di Giacinto"}},
		Usage:       "kairos escrow key agent component",
		Description: ``,
		UsageText:   ``,
		Copyright:   "Ettore Di Giacinto",
		Commands: []*cli.Command{
			{

				Name:        "encrypt",
				Description: "Encrypts a partition",
				Usage:       "Encrypts a partition",
				ArgsUsage:   "kcrypt [--tpm] [--tpm-pcrs] [--public-key-pcrs] LABEL",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "tpm",
						Usage: "Use TPM measurements to lock the partition",
					},
					&cli.StringSliceFlag{
						Name:  "tpm-pcrs",
						Usage: "tpm pcrs to bind to (single measurement) . Only applies when --tpm is also set.",
					},
					&cli.StringSliceFlag{
						Name:  "public-key-pcrs",
						Usage: "public key pcrs to bind to (policy). Only applies when --tpm is also set.",
						Value: cli.NewStringSlice("11"),
					},
				},
				Action: func(c *cli.Context) error {
					var err error
					var out string
					if c.NArg() != 1 {
						return fmt.Errorf("requires 1 arg, the partition label")
					}
					log := types.NewKairosLogger("kcrypt-lock", "info", false)
					if c.Bool("tpm") {
						err = lib.LuksifyMeasurements(c.Args().First(), c.StringSlice("tpm-pcrs"), c.StringSlice("public-key-pcrs"), log)
					} else {
						out, err = lib.Luksify(c.Args().First(), log)
						fmt.Println(out)
					}
					if err != nil {
						return err
					}

					return nil
				},
			},

			{
				Name:        "unlock-all",
				UsageText:   "unlock-all",
				Usage:       "Try to unlock all LUKS partitions",
				Description: "Typically run during initrd to unlock all the LUKS partitions found",
				ArgsUsage:   "kcrypt [--tpm] unlock-all",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "tpm",
						Usage: "Use TPM to unlock the partition",
					},
				},
				Action: func(c *cli.Context) error {
					return lib.UnlockAll(c.Bool("tpm"))
				},
			},
			{

				Name:   "extract-initrd",
				Hidden: true,
				Action: func(c *cli.Context) error {
					if c.NArg() != 2 {
						return fmt.Errorf("requires 3 args. initrd,, dst")
					}
					return lib.ExtractInitrd(c.Args().First(), c.Args().Get(1))
				},
			},
			{
				Name:   "inject-initrd",
				Hidden: true,
				Action: func(c *cli.Context) error {
					if c.NArg() != 3 {
						return fmt.Errorf("requires 3 args. initrd, srcfile, dst")
					}
					return lib.InjectInitrd(c.Args().First(), c.Args().Get(1), c.Args().Get(2))
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
