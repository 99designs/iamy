package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/99designs/iamy/iamy"
	"github.com/fatih/color"
)

type PushCommandInput struct {
	Dir string
}

func PushCommand(ui Ui, input PushCommandInput) {
	yaml := iamy.YamlLoadDumper{
		Dir: input.Dir,
	}
	aws := iamy.AwsFetcher{
		SkipFetchingPolicyAndRoleDescriptions: true,
		ExcludeS3: *excludeS3,
		Debug: ui.Debug,
	}

	allDataFromYaml, err := yaml.Load()
	if err != nil {
		ui.Fatal(err)
		return
	}

	dataFromAws, err := aws.Fetch()
	if err != nil {
		ui.Fatal(err)
		return
	}

	// find the yaml account data that matches the aws account
	for _, dataFromYaml := range allDataFromYaml {
		if dataFromYaml.Account.Id == dataFromAws.Account.Id {
			sync(dataFromYaml, dataFromAws, ui)
			return
		}
	}

	ui.Println("No files found for AWS Account ID " + dataFromAws.Account.Id)
}

func printCommands(prefix string, awsCmds iamy.CmdList, ui Ui) {
	for _, cmd := range awsCmds {
		cmdStr := cmd.String()
		if cmd.IsDestructive() {
			cmdStr = color.RedString(cmdStr)
		}
		ui.Println(prefix + cmdStr)
	}
}

func sync(yamlData iamy.AccountData, awsData *iamy.AccountData, ui Ui) {
	ui.Debug.Printf("Generating sync commands for %s", awsData.Account.String())

	awsCmds := iamy.AwsCliCmdsForSync(awsData, &yamlData)
	if len(awsCmds) == 0 {
		ui.Println("Already up to date")
		return
	}

	ui.Println("Commands to push changes to AWS:\n")

	printCommands("      ", awsCmds, ui)

	if *dryRun {
		ui.Println("Dry-run mode not running aws commands")
		return
	}
	r, err := prompt(fmt.Sprintf("\nRun %d aws commands (%d destructive)? (y/N) ", awsCmds.Count(), awsCmds.CountDestructive()))
	if err != nil {
		ui.Fatal(err)
		return
	}
	if r == "y" {
		for _, c := range awsCmds {
			execCmd(c, ui)
		}
	} else {
		ui.Println("Not running aws commands")
	}
}

func execCmd(c iamy.Cmd, ui Ui) {
	ui.Println("\n>", c)
	cmd := exec.Command(c.Name, c.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		ui.Fatal(err)
		return
	}
}

func prompt(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}
