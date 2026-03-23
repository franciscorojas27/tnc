package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"strings"
)

func exportResults(filename, format string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		data, err := json.MarshalIndent(globalResults, "", "  ")
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	case "csv":
		w := csv.NewWriter(f)
		header := []string{"Target", "IP", "ResolvedName", "SourceAddress", "Interface", "PingRTT", "NetBIOS", "Signing", "OS", "MAC", "Vendor", "Ports"}
		if err := w.Write(header); err != nil {
			return err
		}
		for _, r := range globalResults {
			var b strings.Builder
			for i, p := range r.Ports {
				if i > 0 {
					b.WriteString(";")
				}
				b.WriteString(p.Protocol)
				b.WriteString(":")
				b.WriteString(p.Port)
				b.WriteString(":")
				b.WriteString(p.Status)
				if p.Duration != "" {
					b.WriteString(":" + p.Duration)
				}
			}
			if err := w.Write([]string{r.Target, r.IP, r.ResolvedName, r.SourceAddr, r.Interface, r.PingRTT, r.NetBIOS, r.SMBSigning, r.OS, r.MAC, r.Vendor, b.String()}); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	case "html":
		var b strings.Builder
		b.WriteString("<html><body style='background:#111;color:#eee;font-family:sans-serif'><h2>Scan Report</h2><table border='1' style='border-collapse:collapse;width:100%'><tr>")
		cols := []string{"Target", "IP", "ResolvedName", "SourceAddress", "Interface", "PingRTT", "NetBIOS", "Signing", "OS", "MAC", "Vendor", "Ports"}
		for _, c := range cols {
			b.WriteString(fmt.Sprintf("<th>%s</th>", html.EscapeString(c)))
		}
		b.WriteString("</tr>")
		for _, r := range globalResults {
			b.WriteString("<tr>")
			// prepare ports
			var pStr strings.Builder
			for _, p := range r.Ports {
				pStr.WriteString(html.EscapeString(fmt.Sprintf("%s:%s:%s", p.Protocol, p.Port, p.Status)))
				if p.Duration != "" {
					pStr.WriteString("(" + html.EscapeString(p.Duration) + ")")
				}
				pStr.WriteString(" ")
			}
			vals := []string{r.Target, r.IP, r.ResolvedName, r.SourceAddr, r.Interface, r.PingRTT, r.NetBIOS, r.SMBSigning, r.OS, r.MAC, r.Vendor, pStr.String()}
			for _, v := range vals {
				b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(v)))
			}
			b.WriteString("</tr>")
		}
		b.WriteString("</table></body></html>")
		_, err = f.WriteString(b.String())
		return err
	case "txt", "":
		for _, r := range globalResults {
			if _, err := fmt.Fprintf(f, "Target: %s | IP: %s | Status: %s | OS: %s\n", r.Target, r.IP, r.Status, r.OS); err != nil {
				return err
			}
			for _, p := range r.Ports {
				if _, err := fmt.Fprintf(f, "  Port %s [%s] %s\n", p.Port, p.Protocol, p.Service); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return errors.New("unsupported format: " + format)
	}
}

func runComparison(oldFile string) error {
	data, err := os.ReadFile(oldFile)
	if err != nil {
		return err
	}
	var oldResults []HostResult
	if err := json.Unmarshal(data, &oldResults); err != nil {
		return err
	}
	oldMap := make(map[string]bool, len(oldResults))
	for _, r := range oldResults {
		oldMap[r.IP] = true
	}
	fmt.Println("\n" + Red + "--- NEW DEVICES DETECTED ---" + Reset)
	for _, r := range globalResults {
		if !oldMap[r.IP] {
			fmt.Printf("%s[!] NEW: %s (%s)%s\n", Red, r.IP, r.NetBIOS, Reset)
		}
	}
	return nil
}
