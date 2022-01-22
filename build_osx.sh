set GOOS=darwin
set GOARCH=amd64

go build -ldflags="-s -w" -o builds/osx/grabber_osx_x64 main.go

set GOOS=darwin
set GOARCH=arm64

go build -ldflags="-s -w" -o builds/osx/grabber_osx_arm64 main.go

set GOOS=
set GOARCH=
