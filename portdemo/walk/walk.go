// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Walks the Windows UDP ephemeral-port cursor up to the edge of a
// TCP-excluded port range. Windows hands out ephemeral ports
// sequentially and system-wide, so after this exits, the next few
// UDP-picked ports in ANY process fall inside the excluded range —
// which makes the old UDP-pick-then-TCP-bind test pattern fail
// deterministically instead of once in a few hundred runs.
//
// Run immediately before the test under demonstration:
//
//	go run ./portdemo/walk
//	go test -count=1 -run TestMultiTCPMuxUsage .
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

type portRange struct{ start, end int }

func excludedRanges() []portRange {
	out, err := exec.Command("netsh", "interface", "ipv4", "show",
		"excludedportrange", "protocol=tcp").Output()
	if err != nil {
		fmt.Println("netsh failed:", err)
		return nil
	}
	var ranges []portRange
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		var s, e int
		if n, _ := fmt.Sscanf(strings.TrimSpace(sc.Text()), "%d %d", &s, &e); n == 2 {
			ranges = append(ranges, portRange{s, e})
		}
	}
	return ranges
}

func main() {
	// Only ranges inside the ephemeral pool (49152+) are reachable by the
	// sequential cursor; lower excluded ports are never picked by bind :0.
	var band portRange
	for _, r := range excludedRanges() {
		if r.start >= 49152 {
			band = r
			break
		}
	}
	if band.start == 0 {
		fmt.Println("no TCP excluded range inside the ephemeral pool (49152+);")
		fmt.Println("nothing to aim for on this machine.")
		os.Exit(1)
	}
	fmt.Printf("aiming at excluded range %d-%d\n", band.start, band.end)

	// Stop a few ports short of the range end so the next several picks
	// (the test makes a handful before anything else) land inside it.
	for i := 0; i < 200000; i++ {
		conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
		if err != nil {
			continue
		}
		port := conn.LocalAddr().(*net.UDPAddr).Port
		conn.Close()
		if port >= band.start-3 && port <= band.end-3 {
			fmt.Printf("cursor positioned after %d binds: last UDP port %d\n", i+1, port)
			fmt.Println("run the test NOW (other apps binding UDP will move the cursor)")
			return
		}
	}
	fmt.Println("gave up after 200000 binds without reaching the range")
	os.Exit(1)
}
