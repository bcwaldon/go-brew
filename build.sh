#!/bin/bash

GOOS=linux GOARCH=arm GOARM=7 go build -o build/temp-printer temp.go
