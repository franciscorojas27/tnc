package main

import (
	"net"
	"strconv"
	"strings"
)

func parsePorts(input string) []string {
	seen := make(map[string]struct{})
	ports := make([]string, 0)

	for _, raw := range strings.Split(input, ",") {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}

		if strings.Contains(p, "-") {
			rng := strings.Split(p, "-")
			if len(rng) != 2 {
				continue
			}
			start, errS := strconv.Atoi(strings.TrimSpace(rng[0]))
			end, errE := strconv.Atoi(strings.TrimSpace(rng[1]))
			if errS != nil || errE != nil {
				continue
			}
			if start > end {
				start, end = end, start
			}
			for i := start; i <= end; i++ {
				if i < 1 || i > 65535 {
					continue
				}
				sp := strconv.Itoa(i)
				if _, ok := seen[sp]; ok {
					continue
				}
				seen[sp] = struct{}{}
				ports = append(ports, sp)
			}
			continue
		}

		v, err := strconv.Atoi(p)
		if err != nil || v < 1 || v > 65535 {
			continue
		}
		sp := strconv.Itoa(v)
		if _, ok := seen[sp]; ok {
			continue
		}
		seen[sp] = struct{}{}
		ports = append(ports, sp)
	}

	return ports
}

func parseTargets(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return []string{"localhost"}
	}

	if strings.Contains(input, "/") {
		ip, ipnet, err := net.ParseCIDR(input)
		if err != nil {
			return []string{input}
		}
		var ips []string
		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			ips = append(ips, ip.String())
		}
		if len(ips) > 2 {
			return ips[1 : len(ips)-1]
		}
		return ips
	}

	if strings.Contains(input, "-") {
		parts := strings.Split(input, "-")
		if len(parts) != 2 {
			return []string{input}
		}
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		lastDot := strings.LastIndex(left, ".")
		if lastDot == -1 {
			return []string{input}
		}
		prefix := left[:lastDot+1]
		start, errS := strconv.Atoi(left[lastDot+1:])
		end, errE := strconv.Atoi(right)
		if errS != nil || errE != nil || start < 0 || end < 0 || start > 255 || end > 255 {
			return []string{input}
		}
		if start > end {
			start, end = end, start
		}
		list := make([]string, 0, end-start+1)
		for i := start; i <= end; i++ {
			list = append(list, prefix+strconv.Itoa(i))
		}
		return list
	}

	return []string{input}
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
