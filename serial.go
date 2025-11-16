// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

// Find a serial port device on /dev/ttyACM* or /dev/cu.usbmodem*. This is
// meant to work with a CircuitPython board on macOS or Raspbian.
//
// Note to future me: Don't try to use the /dev/tty.usbmodem* devices on macOS.
// MacOS thinks tty devices are for DCE-initiated incoming calls from a modem
// that asserts DCD. The cu devices with matching numbers are for outbound
// DTE-initiated calls using that device. Use the cu devices if you want to
// avoid wasting time on debugging mysterious blocking I/O.
func serialFindPort() (string, error) {
	patterns := []string{"/dev/ttyACM*", "/dev/cu.usbmodem*"}
	var possibles []string

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return "", err
		}
		possibles = append(possibles, matches...)
	}

	switch len(possibles) {
	case 0:
		return "", errors.New("no serial port found")
	case 1:
		return possibles[0], nil
	default:
		return "", errors.New("too many serial ports")
	}
}

// Configure serial port by shelling out to OS provided stty CLI tool.
// Go doesn't provide serial port or stty support in the standard library, so
// I had a choice here to pull in a dependency from some go community library
// or to use the OS provided stty with Open(). I chose stty + Open().
func serialSttyConfig(port string) error {
	// Try GNU stty -F syntax first.
	cmd := exec.Command("stty", "-F", port, "115200", "clocal", "cread",
		"-crtscts", "cs8", "-hupcl", "-cstopb", "-parenb", "-echo")
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Maybe this is macOS, try it again with the BSD stty -f syntax
	cmd = exec.Command("stty", "-f", port, "115200", "clocal", "cread",
		"-crtscts", "cs8", "-hupcl", "-cstopb", "-parenb", "-echo")
	if err := cmd.Run(); err != nil {
		// WARNING: This will cause the server process to exit
		return fmt.Errorf("stty failed on %s: %w", port, err)
	}

	return nil
}

// Open serial port and begin watching for sensor reports
func serialMonitor(port string, out chan<- string) error {
	if err := serialSttyConfig(port); err != nil {
		return err
	}

	f, err := os.Open(port)
	if err != nil {
		return err
	}
	defer f.Close()

	re := regexp.MustCompile("(LORA|ESPNOW): .*")
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if re.MatchString(line) {
			out <- line
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	log.Printf("Serial port EOF for %s", port)
	return nil
}

// Establish and maintain a serial connection to the serial sensor. If you
// unplug the sensor temporarily, this should re-connect even the OS assigns
// it to a new device file (e.g. ttyACM1 instead of ttyACM0).
func SerialConnect(out chan<- string) {
	for {
		// Find serial port device filename (e.g. /dev/ttyACM0, etc)
		port, err := serialFindPort()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		// Monitor the serial port until there's an EOF or IO error
		log.Printf("Monitoring %v", port)
		if err := serialMonitor(port, out); err != nil {
			log.Printf("%s disconnected: %v", port, err)
		}
		time.Sleep(time.Second)
	}
}
