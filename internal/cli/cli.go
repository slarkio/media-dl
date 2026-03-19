package cli

import (
	"context"
	"flag"

	"github.com/urfave/cli/v3"
)

type DownloadOptions struct {
	URL           string
	OutputDir     string
	Cookie        string
	JSON          bool
	Verbose       bool
	AudioOnly     bool
	ShownotesOnly bool
}

var OutputFlag = &cli.StringFlag{
	Name:    "output",
	Aliases: []string{"o"},
	Usage:   "Output directory",
	Value:   ".",
}

var CookieFlag = &cli.StringFlag{
	Name:    "cookie",
	Aliases: []string{"c"},
	Usage:   "Cookie file path for authenticated requests",
}

var JSONFlag = &cli.BoolFlag{
	Name:    "json",
	Aliases: []string{"j"},
	Usage:   "Output JSON format",
}

var VerboseFlag = &cli.BoolFlag{
	Name:    "verbose",
	Aliases: []string{"v"},
	Usage:   "Verbose debug output",
}

var AudioOnlyFlag = &cli.BoolFlag{
	Name:  "audio-only",
	Usage: "Download audio only, skip shownotes",
}

var ShownotesOnlyFlag = &cli.BoolFlag{
	Name:  "shownotes-only",
	Usage: "Download shownotes only, skip audio",
}

func DownloadCommand(ctx context.Context, cmd *cli.Command) error {
	opts := DownloadOptions{
		URL:           cmd.Args().First(),
		OutputDir:     cmd.String("output"),
		Cookie:        cmd.String("cookie"),
		JSON:          cmd.Bool("json"),
		Verbose:       cmd.Bool("verbose"),
		AudioOnly:     cmd.Bool("audio-only"),
		ShownotesOnly: cmd.Bool("shownotes-only"),
	}

	if opts.URL == "" {
		if opts.JSON {
			printJSONError("url required")
			return flag.ErrHelp
		}
		cli.ShowAppHelp(cmd)
		return flag.ErrHelp
	}

	runner := NewRunner(opts)
	return runner.Run(ctx)
}
