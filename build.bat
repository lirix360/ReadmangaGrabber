@echo off

set GOOS=windows
set GOARCH=amd64

go build -ldflags="-s -w" -o builds/readmanga_grabber_win_x64.exe grabber.go

set GOOS=linux
set GOARCH=amd64

go build -ldflags="-s -w" -o builds/readmanga_grabber_linux_x64 grabber.go

set GOOS=darwin
set GOARCH=amd64

go build -ldflags="-s -w" -o builds/readmanga_grabber_macos_x64 grabber.go

set GOOS=
set GOARCH=