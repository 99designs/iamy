package main

import (
	"strings"

	"github.com/mitchellh/cli"
)

type LoadCommand struct {
	Ui cli.Ui
}

func (c *LoadCommand) Run(args []string) int {
	c.Ui.Error("Not implemented yet")
	return 1
}

func (c *LoadCommand) Help() string {
	helpText := `
Usage: aws-vault store [--profile=default]
  Stores a Access Key Id and Secret Access Key to the vault via interactive prompts.
`
	return strings.TrimSpace(helpText)
}

func (c *LoadCommand) Synopsis() string {
	return "Store credentials to the vault via interactive prompts"
}
