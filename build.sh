#!/bin/sh

export GOOS=windows
export GOARCH=386

go build -ldflags '-s' -o dist/readmanga_grabber_win.exe grabber.go

export GOOS=linux
export GOARCH=386

go build -ldflags '-s' -o dist/readmanga_grabber_linux grabber.go

export GOOS=darwin
export GOARCH=386

go build -ldflags '-s' -o dist/readmanga_grabber_macos grabber.go