#!/bin/bash

env GOOS=linux
env GOARCH=amd64

go build -ldflags="-s -w" -o builds/linux/grabber_linux_x64 main.go

env GOOS=
env GOARCH=
