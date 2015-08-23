package main

import (
	"log"
	"os"

	"github.com/99designs/iamy/Godeps/_workspace/src/github.com/mitchellh/cli"
	"github.com/99designs/iamy/iamy"
)

var (
	Version string
)

func main() {
	ui := &cli.ColoredUi{
		InfoColor: cli.UiColorCyan,
		Ui: &cli.BasicUi{
			Writer: os.Stdout,
			Reader: os.Stdin,
		},
	}

	c := cli.NewCLI("iamy", Version)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"dump-to-yaml": func() (cli.Command, error) {
			return &DumpCommand{
				Ui: ui,
			}, nil
		},
		"generate-sync-cmds": func() (cli.Command, error) {
			return &SyncCommand{
				Ui: ui,
			}, nil
		},
	}

	iamy.Logger = ui.Info

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
