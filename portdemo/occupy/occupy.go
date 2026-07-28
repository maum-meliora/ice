// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Occupies TCP ports just ahead of the UDP ephemeral-port cursor while
// leaving their UDP side free, then holds them for 60 seconds.
//
// Windows hands out ephemeral ports sequentially and system-wide, so while
// this runs, a UDP-picked port in any process is UDP-free but TCP-busy —
// exactly the condition under which the old UDP-pick-then-TCP-bind test
// pattern fails, on any machine, with no excluded port ranges needed.
package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	probe, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		fmt.Println("cursor probe failed:", err)
		return
	}
	cursor := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	var listeners []net.Listener
	// Scan past the target count: some ports in the window are already
	// taken or excluded, and occupation only needs to be dense, not total.
	for port := cursor + 1; port <= cursor+240 && len(listeners) < 120; port++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		listeners = append(listeners, l)
	}
	fmt.Printf("holding %d TCP ports ahead of UDP cursor %d\n", len(listeners), cursor)
	fmt.Println("READY - run the test now; ports release in 60s")
	time.Sleep(60 * time.Second)
	for _, l := range listeners {
		l.Close()
	}
}
