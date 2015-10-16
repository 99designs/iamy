package main

import "github.com/99designs/iamy/iamy"

type PullCommandInput struct {
	Dir       string
	CanDelete bool
}

func PullCommand(ui Ui, input PullCommandInput) {
	data, err := iamy.Aws.Fetch()
	if err != nil {
		ui.Error.Fatal(err)
	}

	iamy.Yaml.Dir = input.Dir
	err = iamy.Yaml.Dump(data, input.CanDelete)
	if err != nil {
		ui.Error.Fatal(err)
	}
}
