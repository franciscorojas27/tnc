package main

import (
	"flag"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Cyan   = "\033[36m"
	Yellow = "\033[33m"
)

func main() {
	comp := flag.String("ComputerName", "localhost", "")
	portsStr := flag.String("Port", "", "")
	trace := flag.Bool("Trace", false, "")
	flag.Parse()

	ips, _ := net.LookupIP(*comp)
	addr := *comp
	if len(ips) > 0 {
		addr = ips[0].String()
	}

	resolvedName := "Unknown"
	names, _ := net.LookupAddr(addr)
	if len(names) > 0 {
		resolvedName = strings.TrimSuffix(names[0], ".")
	}

	sourceAddr := "Unknown"
	ifaceName := "Unknown"
	tempConn, err := net.DialTimeout("udp", net.JoinHostPort(addr, "53"), 1*time.Second)
	if err == nil {
		sourceAddr, _, _ = net.SplitHostPort(tempConn.LocalAddr().String())
		if interfaces, err := net.Interfaces(); err == nil {
			for _, iface := range interfaces {
				addrs, _ := iface.Addrs()
				for _, a := range addrs {
					if strings.Contains(a.String(), sourceAddr) {
						ifaceName = iface.Name
						break
					}
				}
			}
		}
		tempConn.Close()
	}

	fmt.Println("\n" + Cyan + strings.Repeat("-", 45) + Reset)
	fmt.Printf("%-22s : %s\n", "ComputerName", Yellow+*comp+Reset)
	fmt.Printf("%-22s : %s\n", "RemoteAddress", addr)
	fmt.Printf("%-22s : %s\n", "ResolvedName", resolvedName)

	m, v := getMacAndVendor(addr)
	if m != "" {
		fmt.Printf("%-22s : %s\n", "MAC Address", m)
		fmt.Printf("%-22s : %s\n", "Manufacturer", v)
	}

	var pingOut string
	if runtime.GOOS == "windows" {
		out, _ := exec.Command("ping", "-n", "1", "-w", "1000", addr).CombinedOutput()
		pingOut = string(out)
	} else {
		out, _ := exec.Command("ping", "-c", "1", "-W", "1", addr).CombinedOutput()
		pingOut = string(out)
	}

	if strings.Contains(strings.ToUpper(pingOut), "TTL") {
		ttlVal := parseTTL(pingOut)
		osGuess := "Unknown"
		if ttlVal > 0 {
			if ttlVal <= 64 {
				osGuess = "Linux/Unix"
			} else if ttlVal <= 128 {
				osGuess = "Windows"
			} else {
				osGuess = "Network Device"
			}
		}
		fmt.Printf("%-22s : %s (%s) [OS: %s]\n", "PingStatus", Green+"Successful"+Reset, parseRTT(pingOut), Yellow+osGuess+Reset)
	} else {
		fmt.Printf("%-22s : %s\n", "PingStatus", Red+"Failed"+Reset)
	}

	fmt.Printf("%-22s : %s\n", "SourceAddress", sourceAddr)
	fmt.Printf("%-22s : %s\n", "Interface", ifaceName)

	fullPortsStr := *portsStr
	for _, arg := range flag.Args() {
		fullPortsStr += "," + arg
	}

	if fullPortsStr != "" {
		for _, p := range strings.Split(fullPortsStr, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			start := time.Now()
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(addr, p), 2*time.Second)
			if err == nil {
				fmt.Printf("%-22s : %s [%s] (%v)\n", "Port "+p, Green+"Open"+Reset, "TCP", time.Since(start).Truncate(time.Millisecond))
				conn.Close()
			} else {
				fmt.Printf("%-22s : %s [%s]\n", "Port "+p, Red+"Closed"+Reset, "TCP")
			}
		}
	}

	if *trace {
		runTrace(addr)
	}
	fmt.Println(Cyan + strings.Repeat("-", 45) + Reset)
}

func getMacAndVendor(ip string) (string, string) {
	_ = exec.Command("ping", "-n", "1", "-w", "100", ip).Run()
	out, _ := exec.Command("arp", "-a", ip).Output()
	for _, l := range strings.Split(string(out), "\n") {
		if strings.Contains(l, ip) {
			for _, p := range strings.Fields(l) {
				if strings.Count(p, "-") == 5 || strings.Count(p, ":") == 5 {
					mac := strings.ToUpper(strings.ReplaceAll(p, "-", ":"))
					prefix := strings.ReplaceAll(mac[:8], ":", "")
					vendors := map[string]string{
						"00155D": "Microsoft", "000C29": "VMware", "005056": "VMware",
						"B42E99": "Dell", "001122": "HP", "ACDE48": "Apple",
						"28D244": "Samsung", "F0D5BF": "Cisco", "BCAD28": "HP",
						"00000C": "Cisco", "482AD2": "Intel",
					}
					vendor := "Unknown"
					if val, ok := vendors[prefix]; ok {
						vendor = val
					}
					return mac, vendor
				}
			}
		}
	}
	return "", ""
}

func runTrace(target string) {
	fmt.Printf("\n%s%-22s%s\n", Yellow, "Tracing Route:", Reset)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("tracert", "-d", "-h", "10", target)
	} else {
		cmd = exec.Command("traceroute", "-n", "-m", "10", target)
	}
	out, _ := cmd.Output()
	for _, l := range strings.Split(string(out), "\n") {
		t := strings.TrimSpace(l)
		if len(t) > 0 && t[0] >= '1' && t[0] <= '9' {
			fmt.Printf("  %s\n", t)
		}
	}
}

func parseRTT(out string) string {
	out = strings.ToLower(out)
	for _, m := range []string{"tiempo=", "time="} {
		if strings.Contains(out, m) {
			p := strings.Split(out, m)
			val := strings.TrimSpace(p[1])
			if end := strings.Index(val, "ms"); end != -1 {
				return val[:end] + "ms"
			}
			return strings.Fields(val)[0] + "ms"
		}
	}
	return "0ms"
}

func parseTTL(out string) int {
	out = strings.ToLower(out)
	if strings.Contains(out, "ttl=") {
		p := strings.Split(out, "ttl=")
		if len(p) > 1 {
			f := strings.Fields(p[1])
			val, _ := strconv.Atoi(strings.TrimSpace(f[0]))
			return val
		}
	}
	return 0
}
