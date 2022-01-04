@echo off

set GOOS=windows
set GOARCH=amd64

go build -ldflags="-s -w" -o builds/windows/grabber_win_x64.exe main.go

set GOOS=
set GOARCH=
