#!/usr/bin/env bash
# 크로스 컴파일 — macOS(arm64/intel) + Windows(x64) 바이너리 생성
# 사용: ./build.sh  (산출물은 스킬 루트의 ../bin/)
set -e
cd "$(dirname "$0")"
mkdir -p ../bin
echo "빌드 중..."
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o ../bin/hwpxgen-mac-arm64   .
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o ../bin/hwpxgen-mac-intel   .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ../bin/hwpxgen-win-x64.exe .
echo "완료: ../bin/"
ls -lh ../bin/
