package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/99designs/iamy/iamy"
)

type PushCommandInput struct {
	Dir string
}

func PushCommand(ui Ui, input PushCommandInput) {
	iamy.Yaml.Dir = input.Dir
	allDataFromYaml, err := iamy.Yaml.Load()
	if err != nil {
		ui.Fatal(err)
		return
	}

	dataFromAws, err := iamy.Aws.Fetch()
	if err != nil {
		ui.Fatal(err)
		return
	}

	for _, dataFromYaml := range allDataFromYaml {
		if dataFromYaml.Account.Id == dataFromAws.Account.Id {
			sync(dataFromYaml, dataFromAws, ui)
			return
		}
	}

	ui.Println("No files found for AWS Account ID " + dataFromAws.Account.Id)
}

func sync(yamlData iamy.AccountData, awsData *iamy.AccountData, ui Ui) {
	ui.Debug.Printf("Generating sync commands for %s", awsData.Account.String())

	awsCmds := iamy.AwsCliCmdsForSync(awsData, &yamlData)
	if len(awsCmds) == 0 {
		ui.Println("Already up to date")
		return
	}

	ui.Println("Commands to push changes to AWS:\n")
	prefix := "        "
	ui.Println(prefix + strings.Replace(awsCmds.String(), "\n", "\n"+prefix, -1))

	r, err := prompt(fmt.Sprintf("\nRun all aws commands? (y/N) "))
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
