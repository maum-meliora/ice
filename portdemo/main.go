// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Port allocation mechanism demo for Windows.
//
// Measures two things on the machine it runs on:
//  1. old pattern (UDP-pick then TCP-bind, what randomPort did): failure count
//     out of N iterations — nonzero when TCP excluded port ranges exist.
//  2. new pattern (TCP bind :0, what listenerPort enables): failure count and
//     how many granted ports fall inside excluded ranges — expected 0 and 0.
//
// Also prints whether this machine has excluded ranges at all; without
// Hyper-V/WSL2 there may be none, in which case only the race window
// (not the excluded-range mechanism) is observable here.
package main

import (
	"bufio"
	"fmt"
	"net"
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

func inRanges(port int, ranges []portRange) bool {
	for _, r := range ranges {
		if port >= r.start && port <= r.end {
			return true
		}
	}
	return false
}

func main() {
	ranges := excludedRanges()
	fmt.Printf("parsed %d excluded TCP ranges\n", len(ranges))
	if len(ranges) == 0 {
		fmt.Println("NOTE: no excluded ranges on this machine — the excluded-range")
		fmt.Println("mechanism cannot fire here. Enable Hyper-V or WSL2 to get ranges.")
	}

	// 1) A port inside an excluded range: TCP bind must fail, UDP may succeed.
	if len(ranges) > 0 {
		p := ranges[0].start
		_, terr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		fmt.Printf("excluded port %d TCP bind: err=%v\n", p, terr)
		u, uerr := net.ListenPacket("udp4", fmt.Sprintf("127.0.0.1:%d", p))
		fmt.Printf("excluded port %d UDP bind: err=%v\n", p, uerr)
		if u != nil {
			u.Close()
		}
	}

	// 2) Old pattern flake rate on this machine: UDP-pick then TCP-bind.
	const n = 3000
	fails := 0
	for i := 0; i < n; i++ {
		probe, err := net.ListenPacket("udp4", "127.0.0.1:0")
		if err != nil {
			continue
		}
		p := probe.LocalAddr().(*net.UDPAddr).Port
		probe.Close()
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err != nil {
			fails++
		} else {
			l.Close()
		}
	}
	fmt.Printf("old pattern (UDP-pick then TCP-bind): %d/%d binds failed\n", fails, n)

	// 3) New pattern: TCP bind :0 — count grants inside excluded ranges.
	inExcluded, bindFails := 0, 0
	for i := 0; i < n; i++ {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			bindFails++
			continue
		}
		if inRanges(l.Addr().(*net.TCPAddr).Port, ranges) {
			inExcluded++
		}
		l.Close()
	}
	fmt.Printf("new pattern (TCP bind :0): %d/%d failed, %d granted inside excluded ranges\n",
		bindFails, n, inExcluded)
}
