package main

import (
	"fmt"

	"github.com/99designs/iamy/iamy"
)

type PullCommandInput struct {
	Dir       string
	CanDelete bool
}

func PullCommand(ui Ui, input PullCommandInput) {
	aws := iamy.AwsFetcher{
		ExcludeS3: *excludeS3,
		Debug: ui.Debug,
	}
	data, err := aws.Fetch()
	if err != nil {
		ui.Error.Fatal(fmt.Printf("%s", err))
	}

	yaml := iamy.YamlLoadDumper{
		Dir: input.Dir,
	}
	err = yaml.Dump(data, input.CanDelete)
	if err != nil {
		ui.Error.Fatal(err)
	}
}
