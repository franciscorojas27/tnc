package main

import (
	"fmt"
	"strings"
	"sync/atomic"
)

func printResult(res HostResult) {
	fmt.Printf("\n%s%s%s\n", Cyan, strings.Repeat("-", 45), Reset)
	// preserve original visible labels/format from previous tool
	fmt.Printf("%-22s : %s\n", "ComputerName", Yellow+res.Target+Reset)
	fmt.Printf("%-22s : %s\n", "RemoteAddress", res.IP)
	if res.ResolvedName != "" {
		fmt.Printf("%-22s : %s\n", "ResolvedName", res.ResolvedName)
	}
	fmt.Printf("%-22s : %s (%s) [OS: %s]\n", "PingStatus", Green+res.Status+Reset, res.PingRTT, res.OS)
	if res.NetBIOS != "" {
		fmt.Printf("%-22s : %s\n", "NetBIOS", Cyan+res.NetBIOS+Reset)
	}
	if res.SMBSigning != "N/A" {
		fmt.Printf("%-22s : %s\n", "SMB Signing", res.SMBSigning)
	}
	if res.MAC != "" {
		fmt.Printf("%-22s : %s (%s)\n", "MAC/Vendor", res.MAC, res.Vendor)
	}
	// SourceAddress and Interface (as in the original tool)
	fmt.Printf("%-22s : %s\n", "SourceAddress", res.SourceAddr)
	fmt.Printf("%-22s : %s\n", "Interface", res.Interface)
	for _, p := range res.Ports {
		// show status and include duration when available
		if p.Duration != "" {
			if p.Status == "Open" {
				fmt.Printf("%-22s : %s [%s] (%s)\n", "Port "+p.Port, Green+"Open"+Reset, p.Protocol, p.Duration)
			} else {
				fmt.Printf("%-22s : %s [%s]\n", "Port "+p.Port, Red+"Closed"+Reset, p.Protocol)
			}
		} else {
			if p.Status == "Open" {
				fmt.Printf("%-22s : %s [%s] %s\n", "Port "+p.Port, Green+"Open"+Reset, p.Protocol, p.Service)
			} else {
				fmt.Printf("%-22s : %s [%s]\n", "Port "+p.Port, Red+"Closed"+Reset, p.Protocol)
			}
		}
	}
	if res.Trace != "" {
		fmt.Printf("\n%sTrace:%s\n%s", Yellow, Reset, res.Trace)
	}
	fmt.Printf("%s%s%s\n", Cyan, strings.Repeat("-", 45), Reset)
}

func updateProgress() {
	mu.Lock()
	defer mu.Unlock()
	done := int(atomic.LoadInt32(&completed))
	if total <= 0 {
		fmt.Printf("\r%s[%s] %d%% (%d/%d)%s", Yellow, strings.Repeat("░", 50), 100, done, total, Reset)
		return
	}
	pct := (done * 100) / total
	if pct > 100 {
		pct = 100
	}
	fill := pct / 2
	if fill < 0 {
		fill = 0
	}
	if fill > 50 {
		fill = 50
	}
	bar := strings.Repeat("█", fill) + strings.Repeat("░", 50-fill)
	fmt.Printf("\r%s[%s] %d%% (%d/%d)%s", Yellow, bar, pct, done, total, Reset)
}
