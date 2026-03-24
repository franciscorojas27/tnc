package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

func processTarget(target string, opts ScanOptions) HostResult {
	res := HostResult{Target: target, Status: "Down", OS: "Unknown", SMBSigning: "N/A"}
	addr := target
	ips, _ := net.LookupIP(target)
	if len(ips) > 0 {
		addr = ips[0].String()
	}
	res.IP = addr
	if names, err := net.LookupAddr(addr); err == nil && len(names) > 0 {
		res.ResolvedName = strings.TrimSuffix(names[0], ".")
	} else {
		// optional fallback to mDNS discovery (useful on local networks)
		if opts.UseMDNS {
			if mdnsName := mdnsResolveName(addr, 1*time.Second); mdnsName != "" {
				res.ResolvedName = mdnsName
			}
		}
	}

	// determine source address and interface used for connecting
	res.SourceAddr = "Unknown"
	res.Interface = "Unknown"
	d := net.Dialer{Timeout: opts.Timeout}
	if tempConn, err := d.Dial("udp", net.JoinHostPort(addr, "53")); err == nil {
		if s, _, err2 := net.SplitHostPort(tempConn.LocalAddr().String()); err2 == nil {
			res.SourceAddr = s
			// find interface name matching the source IP (exact match, not substring)
			if ifaces, ferr := net.Interfaces(); ferr == nil {
				targetIP := net.ParseIP(s)
				for _, iface := range ifaces {
					addrs, _ := iface.Addrs()
					for _, a := range addrs {
						addrStr := a.String()
						// handle CIDR entries like 172.17.128.1/20
						if strings.Contains(addrStr, "/") {
							if ip, _, err := net.ParseCIDR(addrStr); err == nil {
								if ip.Equal(targetIP) {
									res.Interface = iface.Name
									break
								}
							}
						} else {
							if ip := net.ParseIP(strings.TrimSpace(addrStr)); ip != nil {
								if ip.Equal(targetIP) {
									res.Interface = iface.Name
									break
								}
							}
						}
					}
					if res.Interface != "Unknown" {
						break
					}
				}
			}
		}
		tempConn.Close()
	}

	pOut, isAlive := doPing(addr, opts.Timeout, opts.Force)
	res.PingRTT = parseRTT(pOut)
	// if ping failed, be more persistent and try a small TCP probe set when force is enabled
	if !isAlive && opts.Force {
		if hostAliveByTCP(addr, opts.Timeout, opts.Force) {
			isAlive = true
		}
	}

	if opts.HideDown && !isAlive {
		res.Status = "Hidden"
		return res
	}

	if isAlive {
		res.Status = "Successful"
		res.OS = guessOS(parseTTL(pOut))
		if opts.NetBIOS {
			res.NetBIOS = getNetBIOS(addr, opts.Timeout)
			res.SMBSigning = checkSMBSigning(addr, opts.Timeout)
		}

		var tcpPorts []string
		if opts.PortsStr != "" {
			tcpPorts = parsePorts(opts.PortsStr)
		} else if opts.AllPorts {
			tcpPorts = parsePorts(wellKnown)
		}

		for _, p := range tcpPorts {
			start := time.Now()
			var conn net.Conn
			var err error
			attempts := 1
			if opts.Force {
				attempts = 5
			}
			for i := 0; i < attempts; i++ {
				d := net.Dialer{Timeout: opts.Timeout}
				conn, err = d.Dial("tcp", net.JoinHostPort(addr, p))
				if err == nil {
					break
				}
				if opts.Force {
					time.Sleep(200 * time.Millisecond)
				} else {
					time.Sleep(100 * time.Millisecond)
				}
			}
			dur := time.Since(start)
			if err == nil {
				svc := smartFingerprint(conn, p, target)
				res.Ports = append(res.Ports, PortResult{Port: p, Protocol: "TCP", Status: "Open", Service: svc, Duration: dur.Truncate(time.Millisecond).String()})
				conn.Close()
			} else {
				// record closed if desired with zero duration
				res.Ports = append(res.Ports, PortResult{Port: p, Protocol: "TCP", Status: "Closed", Service: "", Duration: ""})
			}
		}

		for _, p := range parsePorts(opts.UDPStr) {
			if scanUDPPort(addr, p, opts.Timeout, opts.Force) {
				res.Ports = append(res.Ports, PortResult{Port: p, Protocol: "UDP", Status: "Open|Filtered", Service: "Unknown"})
			}
		}

		if opts.Trace {
			res.Trace = runTrace(addr, opts.Timeout)
		}
	}

	if opts.Vendor {
		res.MAC, res.Vendor = getMacAndVendor(addr)
	}
	return res
}

func doPing(addr string, timeout time.Duration, force bool) (string, bool) {
	if timeout <= 0 {
		timeout = 400 * time.Millisecond
	}
	attempts := 1
	if force {
		attempts = 5
	}
	var combinedOut strings.Builder
	for i := 0; i < attempts; i++ {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("ping", "-n", "1", "-w", strconv.Itoa(int(timeout.Milliseconds())), addr)
		} else {
			secs := int(timeout.Seconds())
			if secs < 1 {
				secs = 1
			}
			cmd = exec.Command("ping", "-c", "1", "-W", strconv.Itoa(secs), addr)
		}
		out, _ := cmd.CombinedOutput()
		s := string(out)
		combinedOut.WriteString(s)
		if strings.Contains(strings.ToUpper(s), "TTL") {
			return combinedOut.String(), true
		}
		// stronger backoff between attempts when force is enabled
		if force {
			time.Sleep(200 * time.Millisecond)
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return combinedOut.String(), false
}

func parseRTT(out string) string {
	lo := strings.ToLower(out)
	for _, m := range []string{"tiempo=", "time="} {
		if strings.Contains(lo, m) {
			parts := strings.Split(lo, m)
			if len(parts) > 1 {
				val := strings.TrimSpace(parts[1])
				if end := strings.Index(val, "ms"); end != -1 {
					return strings.TrimSpace(val[:end+2])
				}
				// fallback first field
				return strings.Fields(val)[0]
			}
		}
	}
	return "0ms"
}

func smartFingerprint(c net.Conn, port, host string) string {
	_ = c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if port == "80" || port == "443" || port == "8080" {
		_, _ = fmt.Fprintf(c, "HEAD / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", host)
	}
	buf := make([]byte, 256)
	n, _ := c.Read(buf)
	if n > 0 {
		raw := string(buf[:n])
		if strings.Contains(raw, "Server: ") {
			for _, l := range strings.Split(raw, "\r\n") {
				if strings.HasPrefix(l, "Server: ") {
					return strings.TrimPrefix(l, "Server: ")
				}
			}
		}
		line := strings.Split(raw, "\n")
		if len(line) > 0 {
			return strings.TrimSpace(line[0])
		}
	}
	return "Unknown"
}

func scanUDPPort(ip, port string, timeout time.Duration, force bool) bool {
	attempts := 1
	if force {
		attempts = 5
	}
	for i := 0; i < attempts; i++ {
		d := net.Dialer{Timeout: timeout}
		conn, err := d.Dial("udp", net.JoinHostPort(ip, port))
		if err != nil {
			if force {
				time.Sleep(200 * time.Millisecond)
			} else {
				time.Sleep(100 * time.Millisecond)
			}
			continue
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(timeout))
		_, err = conn.Write([]byte{0x00})
		if err != nil {
			return false
		}
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
		return true
	}
	return false
}

// hostAliveByTCP attempts to detect whether a host is up by trying
// a small set of common TCP ports. Returns true if any connect succeeds.
func hostAliveByTCP(addr string, timeout time.Duration, force bool) bool {
	common := []string{"80", "443", "22", "445", "3389", "8080"}
	attempts := 1
	if force {
		attempts = 3
	}
	for _, p := range common {
		var err error
		for i := 0; i < attempts; i++ {
			d := net.Dialer{Timeout: timeout}
			conn, e := d.Dial("tcp", net.JoinHostPort(addr, p))
			err = e
			if e == nil {
				conn.Close()
				return true
			}
			if force {
				time.Sleep(150 * time.Millisecond)
			}
		}
		_ = err
	}
	return false
}

func getNetBIOS(ip string, timeout time.Duration) string {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("udp", net.JoinHostPort(ip, "137"))
	if err != nil {
		return ""
	}
	defer conn.Close()
	query := []byte("\x80\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x20\x43\x4b\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x41\x00\x00\x21\x00\x01")
	_, _ = conn.Write(query)
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err == nil && n > 72 {
		return strings.TrimSpace(string(buf[57:72]))
	}
	return ""
}

func checkSMBSigning(ip string, timeout time.Duration) string {
	d := net.Dialer{Timeout: timeout * 2}
	conn, err := d.Dial("tcp", net.JoinHostPort(ip, "445"))
	if err != nil {
		return "N/A"
	}
	defer conn.Close()
	negotiate := []byte{0x00, 0x00, 0x00, 0x54, 0xff, 0x53, 0x4d, 0x42, 0x72, 0x00, 0x00, 0x00, 0x00, 0x18, 0x01, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x3f, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x41, 0x00, 0x02, 0x50, 0x43, 0x20, 0x4e, 0x45, 0x54, 0x57, 0x4f, 0x52, 0x4b, 0x20, 0x50, 0x52, 0x4f, 0x47, 0x52, 0x41, 0x4d, 0x20, 0x31, 0x2e, 0x30, 0x00, 0x02, 0x4c, 0x41, 0x4e, 0x4d, 0x41, 0x4e, 0x31, 0x2e, 0x30, 0x00, 0x02, 0x57, 0x49, 0x4e, 0x44, 0x4f, 0x57, 0x53, 0x20, 0x46, 0x4f, 0x52, 0x20, 0x57, 0x4f, 0x52, 0x4b, 0x47, 0x52, 0x4f, 0x55, 0x50, 0x53, 0x20, 0x33, 0x2e, 0x31, 0x61, 0x00, 0x02, 0x4c, 0x4d, 0x31, 0x2e, 0x32, 0x58, 0x30, 0x30, 0x32, 0x00, 0x02, 0x4c, 0x41, 0x4e, 0x4d, 0x41, 0x4e, 0x32, 0x2e, 0x31, 0x00, 0x02, 0x4e, 0x54, 0x20, 0x4c, 0x4d, 0x20, 0x30, 0x2e, 0x31, 0x32, 0x00}
	_, _ = conn.Write(negotiate)
	_ = conn.SetReadDeadline(time.Now().Add(timeout * 2))
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	if n > 43 && (buf[43]&0x08 != 0) {
		return "REQUIRED"
	}
	return "NOT REQUIRED"
}

func getMacAndVendor(ip string) (string, string) {
	out, _ := exec.Command("arp", "-a", ip).Output()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, ip) {
			for _, field := range strings.Fields(line) {
				if strings.Count(field, "-") == 5 || strings.Count(field, ":") == 5 {
					mac := strings.ToUpper(strings.ReplaceAll(field, "-", ":"))
					v := "Unknown"
					if len(mac) >= 8 {
						if val, ok := vendorDB[strings.ReplaceAll(mac[:8], ":", "")]; ok {
							v = val
						}
					}
					return mac, v
				}
			}
		}
	}
	return "", ""
}

// mdnsResolveName queries mDNS for common host service entries and
// attempts to match an entry's address to the given IP. Returns the
// discovered name without trailing dot, or empty string if not found.
func mdnsResolveName(ip string, timeout time.Duration) string {
	entriesCh := make(chan *mdns.ServiceEntry, 16)
	found := make(chan string, 1)

	// collector goroutine
	go func() {
		for e := range entriesCh {
			if e == nil {
				continue
			}
			if e.AddrV4 != nil && e.AddrV4.String() == ip {
				name := strings.TrimSuffix(e.Name, ".")
				select {
				case found <- name:
				default:
				}
				return
			}
			if e.AddrV6 != nil && e.AddrV6.String() == ip {
				name := strings.TrimSuffix(e.Name, ".")
				select {
				case found <- name:
				default:
				}
				return
			}
		}
	}()

	// run lookup for a common host service; many devices advertise
	// as _workstation._tcp on local networks
	go func() {
		// Temporarily silence the standard logger to avoid noisy mdns INFO logs.
		prev := log.Writer()
		log.SetOutput(io.Discard)
		mdns.Lookup("_workstation._tcp", entriesCh)
		log.SetOutput(prev)
		close(entriesCh)
	}()

	select {
	case n := <-found:
		return n
	case <-time.After(timeout):
		return ""
	}
}

func parseTTL(o string) int {
	lo := strings.ToLower(o)
	i := strings.Index(lo, "ttl=")
	if i == -1 {
		return 0
	}
	f := strings.Fields(lo[i+4:])
	if len(f) == 0 {
		return 0
	}
	raw := strings.Trim(f[0], ",; ")
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return v
}

func guessOS(t int) string {
	// Use a conservative heuristic: map observed TTL to likely original TTL
	// only when the observed TTL is very close to a known initial value.
	// Common initial TTLs: 64 (Linux/Unix), 128 (Windows), 255 (Network).
	if t <= 0 {
		return "Unknown"
	}
	// If observed TTL is close (<=4 hops) to an initial TTL, pick that OS.
	initialCandidates := []struct {
		ttl  int
		name string
	}{
		{64, "Linux/IoT"},
		{128, "Windows"},
		{255, "Network Device"},
	}
	for _, c := range initialCandidates {
		if c.ttl >= t && (c.ttl-t) <= 8 { // allow up to ~8 hops difference
			return c.name
		}
	}
	// If no close match, return a cautious label.
	return "Unknown"
}

func runTrace(target string, timeout time.Duration) string {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		wait := int(timeout.Milliseconds())
		if wait < 200 {
			wait = 200
		}
		cmd = exec.Command("tracert", "-d", "-h", "10", "-w", strconv.Itoa(wait), target)
	} else {
		cmd = exec.Command("traceroute", "-n", "-m", "10", "-q", "1", target)
	}
	out, _ := cmd.CombinedOutput()
	return string(out)
}
