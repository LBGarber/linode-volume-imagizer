package main

import (
	"fmt"
	builder2 "github.com/lbgarber/linode-volume-imagizer/builder"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func resolveToken(c *cli.Context) (string, error) {
	envToken, ok := os.LookupEnv("LINODE_TOKEN")
	if ok {
		return envToken, nil
	}

	cliToken := c.String("token")
	if cliToken != "" {
		return cliToken, nil
	}

	return "", fmt.Errorf("failed to find linode token")
}

func cliMain(c *cli.Context) error {
	token, err := resolveToken(c)
	if err != nil {
		return err
	}

	volumeId := c.Int("volume_id")
	region := c.String("region")
	builderType := c.String("builder_type")
	download := c.Bool("download")

	builder := builder2.NewImagizer(token)

	if !download {
		image, err := builder.BuildImage(region, builderType, volumeId)
		if err != nil {
			return err
		}

		log.Printf("Successfully created image:\nID: %v\nLabel: %v\nSize: %v\n", image.ID, image.Label, image.Size)
		return nil
	}

	return builder.DownloadImage(region, builderType, volumeId)
}

func main() {

	app := &cli.App{
		Name: "Linode Volume Imagizer",
		Usage: "Convert Linode Volumes to usable Linode Images ",
		Flags: []cli.Flag {
			&cli.StringFlag{
				Name:        "token",
				Aliases:     []string{"t"},
				Usage:       "Your Linode Personal Access Token.",
			},
			&cli.StringFlag{
				Name:        "region",
				Aliases:     []string{"r"},
				Usage:       "The region in which the volume exists.",
				Value: "us-southeast",
			},
			&cli.StringFlag{
				Name:        "builder_type",
				Aliases:     []string{"type"},
				Usage:       "The type of Instance to use as a builder.",
				Value: 		 "g6-standard-2",
			},
			&cli.IntFlag{
				Name:        "volume_id",
				Aliases:     []string{"v"},
				Usage:       "The ID of the volume to imagize.",
				Required: true,
			},
			&cli.BoolFlag{
				Name: "download",
				Aliases: []string{"d"},
				Usage: "If set, the image will be downloaded locally.",
			},
		},
		Action: cliMain,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}