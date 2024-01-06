#!/bin/bash

dt=$(date '+%Y%m%d%H%M');

echo $dt

env GOOS=linux
env GOARCH=amd64

go build -ldflags="-s -w -X github.com/lirix360/ReadmangaGrabber/config.APPver=$dt" -o builds/linux/grabber_linux_x64 main.go

env GOOS=
env GOARCH=
