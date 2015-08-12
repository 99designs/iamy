package main

import (
	"log"
	"os"

	"github.com/mitchellh/cli"
)

var (
	Version string
)

func main() {
	ui := &cli.BasicUi{
		Writer: os.Stdout,
		Reader: os.Stdin,
	}

	c := cli.NewCLI("iamy", Version)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"dump": func() (cli.Command, error) {
			return &DumpCommand{
				Ui: ui,
			}, nil
		},
		"load": func() (cli.Command, error) {
			return &LoadCommand{
				Ui: ui,
			}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
