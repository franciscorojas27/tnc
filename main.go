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
	comp := flag.String("ComputerName", "localhost", "")
	portsStr := flag.String("Port", "", "")
	allPorts := flag.Bool("all", false, "")
	udpPorts := flag.String("udp", "", "")
	trace := flag.Bool("Trace", false, "")
	hd := flag.Bool("hd", false, "")
	vendor := flag.Bool("v", false, "")
	nb := flag.Bool("netbios", false, "")
	maxWorkers := flag.Int("w", 20, "")
	timeoutMS := flag.Int("timeout", 400, "")
	save := flag.String("Save", "", "")
	format := flag.String("format", "txt", "")
	compare := flag.String("Compare", "", "")
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
			updateProgress()
		}(target)
	}
	wg.Wait()
	fmt.Print(ClearL)

	for _, res := range globalResults {
		printResult(res)
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
