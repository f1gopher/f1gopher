#!/bin/sh

VERSION="1.0.0"

GOOS=darwin GOARCH=amd64 go build -pgo=auto -ldflags="-s -w -X 'main.Version=$VERSION' -X 'main.BuildTime=$(date)'" -o f1gopher
zip f1gopher-mac-amd64.zip ./f1gopher
#rm f1gopher