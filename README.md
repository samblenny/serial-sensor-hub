<!-- SPDX-License-Identifier: MIT -->
<!-- SPDX-FileCopyrightText: Copyright 2025 Sam Blenny -->
# Serial Sensor Hub

**DRAFT: WORK IN PROGRESS**

Hub to gather sensor data over USB serial, log it, chart it, and send
notifications by IRC.


## Installing Go

The sensor hub server is written in the Go programming language. You'll need
the Go compiler tools to build the server binary. If you don't have the Go
build tools, you can download them from [go.dev/dl/](https://go.dev/dl/).

The Go download page has links at the top for macOS on ARM and Linux on Intel.
For Raspberry Pi, scroll down to the big list and find the download link ending
in `.linux-arm64.tar.gz`.

For this project, I'm using go1.25.4 on macOS, Raspberry Pi OS, and Debian:
- go1.25.4.darwin-arm64.pkg   (macOS on Apple Silicon)
- go1.25.4.linux-arm64.tar.gz (Raspberry Pi OS on ARM)
- go1.25.4.linux-amd64.tar.gz (Debian on Intel)

Install instructions are at [go.dev/doc/install](https://go.dev/doc/install).
