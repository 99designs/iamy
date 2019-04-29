package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/blang/semver"
	"gopkg.in/alecthomas/kingpin.v2"
)

const versionTooOldError = `Your version of IAMy (%s) is out of date compared to what the local
project expects. You should upgrade to %s to use this project.`

var (
	Version         string = "dev"
	defaultDir      string
	dryRun          *bool
	versionFileName string = ".iamy-version"
)

type logWriter struct{ *log.Logger }

func (w logWriter) Write(b []byte) (int, error) {
	w.Printf("%s", b)
	return len(b), nil
}

type Ui struct {
	*log.Logger
	Error, Debug *log.Logger
	Exit         func(code int)
}

func main() {
	var (
		debug     = kingpin.Flag("debug", "Show debugging output").Bool()
		pull      = kingpin.Command("pull", "Syncs IAM users, groups and policies from the active AWS account to files")
		pullDir   = pull.Flag("dir", "The directory to dump yaml files to").Default(defaultDir).Short('d').String()
		canDelete = pull.Flag("delete", "Delete extraneous files from destination dir").Bool()
		push      = kingpin.Command("push", "Syncs IAM users, groups and policies from files to the active AWS account")
		pushDir   = push.Flag("dir", "The directory to load yaml files from").Default(defaultDir).Short('d').ExistingDir()
	)
	dryRun = kingpin.Flag("dry-run", "Show what would happen, but don't prompt to do it").Bool()

	kingpin.Version(Version)
	kingpin.CommandLine.Help =
		`Read and write AWS IAM users, policies, groups and roles from YAML files.`

	ui := Ui{
		Logger: log.New(os.Stdout, "", 0),
		Error:  log.New(os.Stderr, "", 0),
		Debug:  log.New(ioutil.Discard, "", 0),
		Exit:   os.Exit,
	}

	cmd := kingpin.Parse()

	if *debug {
		ui.Debug = log.New(os.Stderr, "DEBUG ", log.LstdFlags)
		log.SetFlags(0)
		log.SetOutput(&logWriter{ui.Debug})
	} else {
		log.SetOutput(ioutil.Discard)
	}

	performVersionChecks()

	switch cmd {
	case push.FullCommand():
		PushCommand(ui, PushCommandInput{
			Dir: *pushDir,
		})

	case pull.FullCommand():
		PullCommand(ui, PullCommandInput{
			Dir:       *pullDir,
			CanDelete: *canDelete,
		})
	}
}

func init() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		panic(err)
	}
	defaultDir = filepath.Clean(dir)
}

func performVersionChecks() {
	currentIAMyVersion, _ := semver.Make(Version)
	log.Printf("current version is %s", currentIAMyVersion)

	if _, err := os.Stat(versionFileName); !os.IsNotExist(err) {
		log.Printf("%s found", versionFileName)
		fileBytes, _ := ioutil.ReadFile(versionFileName)
		fileContents := string(fileBytes)

		if fileContents != "" {
			re := regexp.MustCompile(`\d\.\d+\.\d`)
			match := re.FindStringSubmatch(fileContents)
			localDesiredVersion, _ := semver.Make(match[0])
			log.Printf("local project wants version %s", localDesiredVersion)

			// We don't want to notify users if the `Version` is "dev" as it's not
			// actually too old. It could be that they are running non-released
			// versions.
			if Version != "dev" && currentIAMyVersion.LE(localDesiredVersion) {
				fmt.Printf(versionTooOldError, currentIAMyVersion, localDesiredVersion)
				os.Exit(1)
			}
		}
	}
}
