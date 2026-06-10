@echo off
REM 크로스 컴파일 — Windows에서 실행. macOS+Windows 바이너리 생성 (산출물은 스킬 루트의 ..\bin\)
cd /d "%~dp0"
if not exist ..\bin mkdir ..\bin
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -o ..\bin\hwpxgen-win-x64.exe .
set GOOS=darwin
set GOARCH=arm64
go build -o ..\bin\hwpxgen-mac-arm64 .
set GOOS=darwin
set GOARCH=amd64
go build -o ..\bin\hwpxgen-mac-intel .
echo 완료: ..\bin\
dir ..\bin
