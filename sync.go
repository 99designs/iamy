package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/iamy/Godeps/_workspace/src/github.com/mitchellh/cli"
	"github.com/99designs/iamy/iamy"
)

type SyncCommand struct {
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

func (c *SyncCommand) Run(args []string) int {
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
	iamy.Yaml.Dir = dir
	dataFromYaml, err := iamy.Yaml.Load()
	if err != nil {
		c.Ui.Error(err.Error())
		return 3
	}

	// load data from AWS
	dataFromAws, err := iamy.Aws.Fetch()
	if err != nil {
		c.Ui.Error(err.Error())
		return 4
	}

	for _, y := range dataFromYaml {
		if y.Account.Id == dataFromAws.Account.Id {
			c.Ui.Info(fmt.Sprintf("Generating sync commands for %s", dataFromAws.Account.String()))
			awsCmds := iamy.AwsCliCmdsForSync(dataFromAws, &y)
			c.Ui.Output(strings.Join(awsCmds, "\n"))
		}
	}

	return 0
}

func (c *SyncCommand) Help() string {
	helpText := `
Usage: iamy dump [-dir <output dir>]
  Loads users, groups and policies from yaml files and generates aws cli commands to sync with AWS
`
	return strings.TrimSpace(helpText)
}

func (c *SyncCommand) Synopsis() string {
	return "Loads users, groups and policies from yaml files and generates aws cli commands to sync with AWS"
}
