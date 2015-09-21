package main

import (
	"strings"

	"github.com/99designs/iamy/iamy"
)

type LoadCommandInput struct {
	Dir string
}

func LoadCommand(ui Ui, input LoadCommandInput) {
	iamy.Yaml.Dir = input.Dir
	dataFromYaml, err := iamy.Yaml.Load()
	if err != nil {
		ui.Fatal(err)
	}

	dataFromAws, err := iamy.Aws.Fetch()
	if err != nil {
		ui.Fatal(err)
	}

	for _, y := range dataFromYaml {
		if y.Account.Id == dataFromAws.Account.Id {
			ui.Debug.Printf("Generating sync commands for %s", dataFromAws.Account.String())
			awsCmds := iamy.AwsCliCmdsForSync(dataFromAws, &y)
			ui.Println(strings.Join(awsCmds, "\n"))
		}
	}
}
