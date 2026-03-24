package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	comp := flag.String("computerName", "localhost", "Host, IP, or CIDR range to scan (default 'localhost')")
	portsStr := flag.String("port", "", "Comma-separated TCP port(s) to scan (e.g. '80,443')")
	allPorts := flag.Bool("all", false, "Scan predefined well-known ports")
	udpPorts := flag.String("udp", "", "Comma-separated UDP port(s) to scan")
	trace := flag.Bool("trace", false, "Perform traceroute to discovered hosts")
	hd := flag.Bool("hd", false, "Hide hosts that do not respond (show only 'up')")
	mdnsFlag := flag.Bool("mdns", false, "Use mDNS as a fallback when hostname resolution fails")
	force := flag.Bool("force", false, "Aggressively retry hosts/ports to be more persistent")
	quiet := flag.Bool("quiet", false, "Suppress console output and progress (quiet mode)")
	vendor := flag.Bool("v", false, "Obtain MAC address and vendor via ARP")
	nb := flag.Bool("netbios", false, "Use NetBIOS to resolve name and check SMB signing")
	maxWorkers := flag.Int("w", 20, "Maximum number of concurrent workers")
	timeoutMS := flag.Int("timeout", 400, "Operation timeout in milliseconds (min 100)")
	save := flag.String("save", "", "File path to save results")
	format := flag.String("format", "txt", "Export format: txt|json|csv (default 'txt')")
	compare := flag.String("compare", "", "File or dataset to compare results against")
	flag.Parse()

	if *maxWorkers < 1 {
		*maxWorkers = 1
	}
	if *timeoutMS < 100 {
		*timeoutMS = 100
	}

	targets := parseTargets(*comp)
	total = len(targets)
	atomic.StoreInt32(&completed, 0)
	mu.Lock()
	globalResults = nil
	mu.Unlock()

	opts := ScanOptions{
		PortsStr: strings.TrimSpace(*portsStr),
		UDPStr:   strings.TrimSpace(*udpPorts),
		AllPorts: *allPorts,
		Trace:    *trace,
		HideDown: *hd,
		UseMDNS:  *mdnsFlag,
		Force:    *force,
		Quiet:    *quiet,
		Vendor:   *vendor,
		NetBIOS:  *nb,
		Timeout:  time.Duration(*timeoutMS) * time.Millisecond,
	}

	sem := make(chan struct{}, *maxWorkers)
	var wg sync.WaitGroup

	fmt.Printf("%s[*] Scanning %d targets...%s\n", Cyan, total, Reset)

	for _, target := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(t string) {
			defer wg.Done()
			defer func() { <-sem }()
			res := processTarget(t, opts)
			mu.Lock()
			if res.Status != "Hidden" {
				globalResults = append(globalResults, res)
			}
			mu.Unlock()
			atomic.AddInt32(&completed, 1)
			if !opts.Quiet {
				updateProgress()
			}
		}(target)
	}
	wg.Wait()
	fmt.Print(ClearL)

	if !opts.Quiet {
		for _, res := range globalResults {
			printResult(res)
		}
	}

	if *compare != "" {
		if err := runComparison(*compare); err != nil {
			fmt.Printf("%s[!] Compare error: %v%s\n", Red, err, Reset)
		}
	}
	if *save != "" {
		if err := exportResults(*save, *format); err != nil {
			fmt.Printf("%s[!] Export error: %v%s\n", Red, err, Reset)
		}
	}
	fmt.Println("\n" + Green + "✔ Scan Completed." + Reset)
}
