// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"log"
)

func main() {
	reportChan := make(chan string)

	go SerialConnect(reportChan)

	for report := range reportChan {
		log.Print(report)
	}
}
