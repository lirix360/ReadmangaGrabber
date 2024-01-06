@echo off

for /f "usebackq skip=1 tokens=1-6" %%g in (`wmic Path Win32_LocalTime Get Day^,Hour^,Minute^,Month^,Second^,Year ^| findstr /r /v "^$"`) do (
  set _day=00%%g
  set _hours=00%%h
  set _minutes=00%%i
  set _month=00%%j
  set _year=%%l
)
set _month=%_month:~-2%
set _day=%_day:~-2%
set _hh=%_hours:~-2%
set _mm=%_minutes:~-2%
set _date=%_year%%_month%%_day%%_hh%%_mm%

echo %_date%

set GOOS=windows
set GOARCH=amd64

go build -ldflags="-s -w -X github.com/lirix360/ReadmangaGrabber/config.APPver=%_date%" -o builds/windows/grabber_win_x64.exe main.go

set GOOS=
set GOARCH=
