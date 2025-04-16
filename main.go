package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/kairos-io/kairos-sdk/types"
	"github.com/kairos-io/kcrypt/pkg/lib"
	"github.com/urfave/cli/v3"
)

var Version = "v0.0.0-dev"
var GitCommit = "none"

func main() {
	log := types.NewKairosLogger("kcrypt", "info", false)
	app := &cli.Command{
		Name:        "kairos-kcrypt",
		Authors:     []any{"Ettore Di Giacinto"},
		Usage:       "kairos escrow key agent component",
		Description: "",
		UsageText:   "",
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
						Usage: "tpm pcrs to bind to (single measurement). Only applies when --tpm is also set.",
					},
					&cli.StringSliceFlag{
						Name:  "public-key-pcrs",
						Usage: "public key pcrs to bind to (policy). Only applies when --tpm is also set.",
						Value: []string{"11"},
					},
				},
				Action: func(_ context.Context, c *cli.Command) error {
					if c.NArg() != 1 {
						return fmt.Errorf("requires 1 arg, the partition label")
					}

					var err error
					if c.Bool("tpm") {
						err = lib.LuksifyMeasurements(c.Args().First(), c.StringSlice("tpm-pcrs"), c.StringSlice("public-key-pcrs"), log)
					} else {
						out, err := lib.Luksify(c.Args().First(), log)
						if err == nil {
							fmt.Println(out)
						}
					}
					return err
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
				Action: func(_ context.Context, c *cli.Command) error {
					return lib.UnlockAllWithLogger(c.Bool("tpm"), log)
				},
			},
			{
				Name:        "version",
				UsageText:   "version",
				Usage:       "Prints the version",
				Description: "Prints the version",
				ArgsUsage:   "kcrypt version",
				Action: func(_ context.Context, _ *cli.Command) error {
					log.Logger.Info().Str("commit", GitCommit).Str("goversion", runtime.Version()).Str("version", Version).Msg("Kcrypt")
					return nil
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
