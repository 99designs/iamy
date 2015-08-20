package main

import (
	"flag"
	"os"
	"strings"

	"github.com/99designs/iamy/loaddumper"
	"github.com/mitchellh/cli"
)

type LoadCommand struct {
	Ui cli.Ui
}

func (c *LoadCommand) Run(args []string) int {
	var dir string
	flagSet := flag.NewFlagSet("dump", flag.ContinueOnError)
	flagSet.StringVar(&dir, "dir", "", "Directory to read files from")
	flagSet.Usage = func() { c.Ui.Output(c.Help()) }

	if err := flagSet.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			c.Ui.Error(err.Error())
			return 2
		}
	}

	// load yaml from dir
	loaddumper.Yaml.Dir = dir
	_, err := loaddumper.Yaml.Load()
	if err != nil {
		c.Ui.Error(err.Error())
		return 4
	}

	// roundtrip
	// loaddumper.Yaml.Dump(data)

	return 1
}

func (c *LoadCommand) Help() string {
	helpText := `
Usage: iamy dump [-dir <output dir>]
  Loads users, groups and policies from yaml files and outputs commands to sync with AWS
`
	return strings.TrimSpace(helpText)
}

func (c *LoadCommand) Synopsis() string {
	return "Loads users, groups and policies from yaml files and outputs commands to sync with AWS"
}
