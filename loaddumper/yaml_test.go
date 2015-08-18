package loaddumper

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)
import "os"

func readDir(path string) map[string][]byte {
	files := map[string][]byte{}
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		if !info.IsDir() {
			files[info.Name()], err = ioutil.ReadFile(p)
			if err != nil {
				panic(err)
			}
		}

		return nil
	})

	return files
}

func newTmpDir() string {
	testdir, err := ioutil.TempDir("", "loaddumpertest")
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("Creating tmp dir", testdir)
	return testdir
}

func TestRoundTrip(t *testing.T) {
	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err.Error())
	}

	Yaml.Dir = filepath.Join(d, "testdata")

	accountData, err := Yaml.Load()
	if err != nil {
		t.Fatal(err.Error())
	}

	testdir := newTmpDir()
	Yaml.Dir = testdir
	err = Yaml.Dump(accountData)
	if err != nil {
		t.Fatal(err.Error())
	}

	yamlDirData := readDir(Yaml.Dir)
	testdirData := readDir(testdir)
	eq := reflect.DeepEqual(yamlDirData, testdirData)
	if !eq {
		t.Error("Directory contents are not equal")
	}
}
