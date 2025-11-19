// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"regexp"
	"strings"
	"time"
)

// Send string to IRC server.
func ircSend(conn net.Conn, msg string) error {
	_, err := conn.Write([]byte(msg + "\r\n"))
	if err != nil {
		log.Printf("WARN: IRC send error: %v", err)
	}
	return err
}

func ircNextBackoff(delay, max time.Duration) time.Duration {
	// Increase current delay by 0.5 to 1.5 of its current value
	delay += time.Duration(float64(delay) * (0.5 + rand.Float64()))
	if delay <= max {
		return delay
	}
	return max
}

// Forward messages from input channel to the configured IRC server
func IRCBot(ctx context.Context, cfg *ServerConfig, in <-chan string) {
	// Regex for parsing IRC lines: {prefix, command, params}
	ircLineRE := regexp.MustCompile(
		`^((:\S+)\s+)?` + // optional prefix (match group 2)
			`(\S+)\s*` + // command or number (match group 3)
			`(.*)`) // params (match group 4)
	// Regex for identifying numeric IRC commands
	numericCmdRE := regexp.MustCompile(`^\d{3}$`)

	// These are for keeping track of connection retry backoff delay
	baseDelay := 3 * time.Second
	maxDelay := 10 * time.Minute
	connDelay := baseDelay

	// Declaring these here makes it possibe to have error handlers trigger an
	// automatic connection close at the top of the next loop iteration by
	// using continue
	var conn net.Conn = nil
	var err error

	// Loop forever with auto-reconnect using polite exponential backoff delay
ConnectLoop:
	for {
		// Auto-close the connection so error handlers can just do a continue
		if conn != nil {
			log.Print("INFO: Closing IRC connection")
			conn.Close()
			conn = nil
		}

		// Always start with a delay before attempting to connect
		time.Sleep(connDelay)

		// Connect or shutdown
		select {
		case <-ctx.Done():
			// Handle shutdown signal
			log.Print("INFO: Shutting down IRCBot (not connected)")
			return
		default:
			// Connect
			log.Printf("INFO: IRC Connecting to %s", cfg.Server)
			conn, err = net.Dial("tcp", cfg.Server)
			if err != nil {
				log.Printf("WARN: IRC connection failed: %v", err)
				conn = nil
				// Increase delay time used by the sleep at top of ConnectLoop
				connDelay = ircNextBackoff(connDelay, maxDelay)
				continue ConnectLoop
			}
			connDelay = baseDelay
		}

		// Send messages to register nick and join channel
		if err = ircSend(conn, "NICK "+cfg.Nick); err != nil {
			continue ConnectLoop
		}
		if err = ircSend(conn, "USER "+cfg.Nick+" 0 * :"+cfg.Nick); err != nil {
			continue ConnectLoop
		}
		if err = ircSend(conn, "JOIN "+cfg.Channel); err != nil {
			continue ConnectLoop
		}

		// Begin conversation with IRC server
		scanner := bufio.NewScanner(conn)

		// Channel for incoming lines from the IRC server connection
		lineChan := make(chan string, 64)

		// Set up the scanner in its own goroutine so we can use its output
		// more easily in a select alongside of <-ctx and <-in
		go func() {
			defer close(lineChan)
			for scanner.Scan() {
				select {
				case lineChan <- scanner.Text():
				case <-ctx.Done():
					log.Printf("DEBUG: IRC scanner got <-ctx.Done()")
					return
				}
			}
			if err := scanner.Err(); err != nil {
				log.Printf("WARN: IRC scanner failed: %v", err)
			}
		}()

		// Connected Input Loop
		registered := false
		joined := false
	InputLoop:
		for {
			// Select between all the input sources
			select {
			case <-ctx.Done():
				// Handle shutdown signal
				log.Print("INFO: Shutting down IRCBot (connected)")
				conn.Close()
				return
			case msg := <-in:
				// Handle a message from the input channel by setting the
				// topic of the configured channel
				if registered && joined {
					ircMsg := fmt.Sprintf("TOPIC %s :%s", cfg.Channel, msg)
					if err := ircSend(conn, ircMsg); err != nil {
						continue ConnectLoop
					}
				}
			case line, ok := <-lineChan:
				// Handle a line from the IRC server
				if !ok {
					log.Print("INFO: IRC connection closed by server")
					continue ConnectLoop
				}

				// Separate line into {prefix, command, params}
				matches := ircLineRE.FindStringSubmatch(line)
				if matches == nil {
					continue InputLoop
				}
				prefix := matches[2] // optional, may be empty
				command := matches[3]
				params := matches[4]

				// Respond according to the command
				switch command {
				case "001": // Welcome message (registration worked)
					log.Printf("IRC: %v %v", command, params)
					registered = true

				case "002": // Ignore these boring startup messsages
				case "003":
				case "004":
				case "005":
				case "250":
				case "251":
				case "252":
				case "254":
				case "255":
				case "265":
				case "266":
				case "333":
				case "353":
				case "366":

				case "433": // Nick in use, reconnect after a delay
					log.Printf("IRC: %s", line)
					connDelay = ircNextBackoff(connDelay, maxDelay)
					continue ConnectLoop

				case "442": // Not on channel (kicked by prankster?)
					log.Printf("IRC: %s", line)
					if strings.HasPrefix(params, cfg.Nick+" "+cfg.Channel) {
						log.Printf("INFO: IRC 442 not in channel")
						continue ConnectLoop
					}

				case "JOIN": // Might be our JOIN or might be somebody else's
					log.Printf("IRC: %s", line)
					nickMatch := strings.HasPrefix(prefix, ":"+cfg.Nick+"!")
					chanMatch := strings.HasPrefix(params, ":"+cfg.Channel)
					if nickMatch && chanMatch {
						log.Printf("INFO: IRC %s joined %s", cfg.Nick,
							cfg.Channel)
						joined = true
					}

				case "PING": // Reply with PONG to keep connection alive
					log.Printf("IRC: %s", line)
					ircSend(conn, "PONG "+params)

				default:
					if numericCmdRE.MatchString(command) {
						// omit prefix from initial connection messages
						log.Printf("IRC: %v %v", command, params)
					} else {
						log.Printf("IRC: %s", line)
					}
				}
			} // end select (<-ctx, <-in, <-lineChan)
		} // end InputLoop
	} // end ConnectLoop
}
