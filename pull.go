package main

import (
	"fmt"

	"github.com/99designs/iamy/iamy"
)

type PullCommandInput struct {
	Dir                  string
	CanDelete            bool
	HeuristicCfnMatching bool
	SkipCfnTagged        bool
}

func PullCommand(ui Ui, input PullCommandInput) {
	aws := iamy.AwsFetcher{Debug: ui.Debug, HeuristicCfnMatching: input.HeuristicCfnMatching, SkipCfnTagged: input.SkipCfnTagged}
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
