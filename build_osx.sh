#!/bin/bash

export GOOS=darwin
export GOARCH=amd64

go build -ldflags="-s -w" -o builds/osx/grabber_osx_x64 main.go

export GOOS=darwin
export GOARCH=arm64

go build -ldflags="-s -w" -o builds/osx/grabber_osx_arm64 main.go

export GOOS=
export GOARCH=
