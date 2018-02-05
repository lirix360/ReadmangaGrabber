#!/bin/sh

export GOOS=windows
export GOARCH=amd64

go build -ldflags '-s' -o dist/readmanga_grabber_win_x64.exe grabber.go

export GOOS=linux
export GOARCH=amd64

go build -ldflags '-s' -o dist/readmanga_grabber_linux_x64 grabber.go

export GOOS=darwin
export GOARCH=amd64

go build -ldflags '-s' -o dist/readmanga_grabber_macos_x64 grabber.go

export GOOS=
export GOARCH=