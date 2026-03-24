package main

import "time"

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Cyan   = "\033[36m"
	Yellow = "\033[33m"
	ClearL = "\033[2K\r"
)

type PortResult struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	Status   string `json:"status"`
	Service  string `json:"service"`
	Duration string `json:"duration"`
}

type HostResult struct {
	Target       string       `json:"target"`
	IP           string       `json:"ip"`
	ResolvedName string       `json:"resolved_name"`
	NetBIOS      string       `json:"netbios"`
	SMBSigning   string       `json:"smb_signing"`
	OS           string       `json:"os"`
	Status       string       `json:"ping_status"`
	MAC          string       `json:"mac"`
	Vendor       string       `json:"vendor"`
	Ports        []PortResult `json:"ports"`
	Trace        string       `json:"trace"`
	PingRTT      string       `json:"ping_rtt"`
	SourceAddr   string       `json:"source_addr"`
	Interface    string       `json:"interface"`
}

type ScanOptions struct {
	PortsStr string
	UDPStr   string
	AllPorts bool
	Trace    bool
	HideDown bool
	UseMDNS  bool
	Force    bool
	Quiet    bool
	Vendor   bool
	NetBIOS  bool
	Timeout  time.Duration
}

var (
	wellKnown = "21,22,23,25,53,80,110,143,443,445,3306,3389,8080"
	vendorDB  = map[string]string{
		"00155D": "Microsoft",
		"000C29": "VMware",
		"B42E99": "Dell",
		"001122": "HP",
		"D897BA": "Hikvision",
		"245EBE": "Ubiquiti",
	}
)
