export GO111MODULE=on
VERSION="$(shell git describe --tags --candidates=1 --dirty)+envato"
FLAGS=-X main.Version=$(VERSION) -s -w

# To create a new release:
#  $ git tag vx.x.x
#  $ git push --tags
#  $ make clean
#  $ make release     # this will create 3 binaries in ./bin
#
#  Next, go to https://github.com/99designs/iamy/releases/new
#  - select the tag version you just created
#  - Attach the binaries from ./bin/*

release: bin/iamy-linux-amd64 bin/iamy-darwin-amd64 bin/iamy-windows-386.exe

bin/iamy-linux-amd64:
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o $@ -ldflags="$(FLAGS)" .

bin/iamy-darwin-amd64:
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o $@ -ldflags="$(FLAGS)" .

bin/iamy-windows-386.exe:
	@mkdir -p bin
	GOOS=windows GOARCH=386 go build -o $@ -ldflags="$(FLAGS)" .

clean:
	rm -f bin/*
