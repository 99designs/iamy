package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	Version    string
	defaultDir string
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
		debug   = kingpin.Flag("debug", "Show debugging output").Bool()
		dump    = kingpin.Command("dump", "Dumps users, groups and policies to files")
		dumpDir = dump.Flag("dir", "The directory to dump yaml files to").Default(defaultDir).Short('d').String()
		load    = kingpin.Command("load", "")
		loadDir = load.Flag("dir", "The directoy to load yaml files from").Default(defaultDir).Short('d').ExistingDir()
	)

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

	switch cmd {
	case load.FullCommand():
		LoadCommand(ui, LoadCommandInput{
			Dir: *loadDir,
		})

	case dump.FullCommand():
		DumpCommand(ui, DumpCommandInput{
			Dir: *dumpDir,
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
