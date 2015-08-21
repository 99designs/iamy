package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/iamy/Godeps/_workspace/src/github.com/mitchellh/cli"
	"github.com/99designs/iamy/loaddumper"
)

type LoadCommand struct {
	Ui cli.Ui
}

func getDirOrDefault(dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}

	return filepath.Clean(dir), nil
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

	dir, err := getDirOrDefault(dir)
	if err != nil {
		c.Ui.Error(err.Error())
		return 2
	}

	// load data from yaml
	loaddumper.Yaml.Dir = dir
	dataFromYaml, err := loaddumper.Yaml.Load()
	if err != nil {
		c.Ui.Error(err.Error())
		return 3
	}

	// roundtrip
	// loaddumper.Yaml.Dump(dataFromYaml)

	// load data from AWS
	dataFromAws, err := loaddumper.Aws.Fetch()
	if err != nil {
		c.Ui.Error(err.Error())
		return 4
	}

	awsCmds := loaddumper.AwsCliCmdsForSync(dataFromAws[0], dataFromYaml[0])
	c.Ui.Info(strings.Join(awsCmds, "\n"))

	return 0
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
