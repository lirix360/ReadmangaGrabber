#!/bin/bash

set GOOS=linux
set GOARCH=amd64

go build -ldflags="-s -w" -o builds/linux/grabber_linux_x64 main.go

set GOOS=
set GOARCH=
