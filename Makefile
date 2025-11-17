# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: Copyright 2025 Sam Blenny

.PHONY: run test clean
SRC_FILES=go.mod irc.go logger.go main.go reports.go serial.go

serial-sensor-hub: Makefile $(SRC_FILES)
	@go build -buildvcs=false -ldflags "-s -w" -trimpath

run: serial-sensor-hub
	@./serial-sensor-hub

test:
	go test

clean:
	go clean
