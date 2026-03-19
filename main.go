package main

import (
	"context"
	"os"

	cli "github.com/slarkio/media-dl/internal/cli"
	urfave "github.com/urfave/cli/v3"
)

func main() {
	app := &urfave.Command{
		Name:  "media-dl",
		Usage: "Media downloader CLI for YouTube, Bilibili, and Xiaoyuzhou - extracts audio and converts shownotes images to markdown",
		ArgsUsage: "<url>",
		Flags: []urfave.Flag{
			cli.OutputFlag,
			cli.CookieFlag,
			cli.JSONFlag,
			cli.VerboseFlag,
			cli.AudioOnlyFlag,
			cli.ShownotesOnlyFlag,
		},
		Action: cli.DownloadCommand,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
