package main

import "github.com/99designs/iamy/iamy"

type DumpCommandInput struct {
	Dir       string
	CanDelete bool
}

func DumpCommand(ui Ui, input DumpCommandInput) {
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
