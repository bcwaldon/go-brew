#!/bin/bash

GOOS=linux GOARCH=arm GOARM=7 go build -o build/temp-printer DS18B20/temp.go
