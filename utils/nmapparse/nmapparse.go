/*
   Raven Network Discovery and Monitoring
   Copyright (C) 2019 John{at}Orthoefer{dot}org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.

*/
package main

import (
	"./license"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
  "sort"
  "github.com/go-ini/ini"
	"io/ioutil"
	"log"
	"net"
  "os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Structure to keep the nmap output XML into
var nmap NmapInfo

// nmap Header XML
type NmapInfo struct {
	Scanner       string       `xml:"scanner,attr"`
	Args          string       `xml:"args,attr"`
	Start         int64        `xml:"start,attr"`
	StartS        string       `xml:"startstr,attr"`
	version       string       `xml:"version,attr"`
	xmloutputvers string       `xml:"xmloutputversion,attr"`
	Info          InfoStruct   `xml:"scaninfo"`
	Host          []HostStruct `xml:"host"`
}

type InfoStruct struct {
	Type  string `xml:"type,attr"`
	Proto string `xml:"protocol,attr"`
	count string `xml:"numservices,attr"`
	svrs  string `xml:"services,attr"`
}

// nmap Host (per host XML)
type HostStruct struct {
  StartTime int64            `xml:"starttime,attr"`
  EndTime   int64            `xml:"endtime,attr"`
  Status    StatusStruct     `xml:"status"`
  Addr      []AddrStruct     `xml:"address"`
  Hostname  []HostnameStruct `xml:"hostnames>hostname"`
  Ports     []PortStruct     `xml:"ports>port"`
  OSInfo    OSStruct         `xml:"os>osmatch"`
}

type StatusStruct struct {
	State     string `xml:"state,attr"`
	Reason    string `xml:"reason,attr"`
	Reasonttl int    `xml:"reason_ttl,attr"`
}
type AddrStruct struct {
	Addr string `xml:"addr,attr"`
	Type string `xml:"addrtype,attr"`
}
type HostnameStruct struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}
type OSStruct struct {
  Name    string        `xml:"name,attr"`
  Acc     string        `xml:"accuracy"`
}
type PortStruct struct {
  Protocol string        `xml:"protocol,attr"`
  PortID   int           `xml:"portid,attr"`
  State    StatusStruct  `xml:"state"`
  Service  ServiceStruct `xml:"service"`
}
type ServiceStruct struct {
	Name   string `xml:"name,attr"`
	Method string `xml:"method,attr"`
	Conf   int    `xml:"conf,attr"`
}

type HostJSON struct {
	Name     string    `json:"name"`
	Hostname string    `json:"hostname"`
	Enabled  bool      `json:"enabled"`
	DHCP     bool      `json:"dhcp"`
	IPv4     string    `json:"ipv4"`
	When     time.Time `json:"lastseen"`
	Ports    []int     `json:"ports"`
  OS       string    `json:"OSName"`
}

func runExternal(prog string, args ...string) (int, string) {

	var out bytes.Buffer

	cmd := exec.Command(prog, args...)

	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd.Start: %v")
	}

	rtnExit := 0
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			// This works on both Unix and Windows. Although package
			// syscall is generally platform dependent, WaitStatus is
			// defined for both Unix and Windows and in both cases has
			// an ExitStatus() method with the same signature.
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				rtnExit = status.ExitStatus()
				log.Printf("Exit Status: %d", rtnExit)
			}
		} else {
			log.Fatalf("cmd.Wait: %v", err)
		}
	}
	return rtnExit, out.String()
}

func getHostInfo(h HostStruct, dl, dh int) (name, hn, hi string, dhcp bool) {
	if len(h.Hostname) > 0 {
		hn = h.Hostname[0].Name
	}

	if len(h.Addr) > 0 {
		hi = h.Addr[0].Addr
	}

	octets := strings.Split(hi, ".")
	lsv, _ := strconv.Atoi(octets[3])
	if hn == "" {
		name = fmt.Sprintf("NO-Name-%d", lsv)
    dhcp = false
	} else {
    s := strings.Split(hn, ".")
    name = s[0]
	  dhcp = (lsv >= dl) && (lsv <= dh)
  }
	return name, hn, hi, dhcp
}

func findLocal() string {
	addr, _ := net.InterfaceAddrs()
	for _, j := range addr {
		ip, _, err := net.ParseCIDR(j.String())
		if err != nil {
			log.Printf("ParseCIDR failed")
			continue
		}
		if ip.IsGlobalUnicast() {
			log.Printf("Found %v", j.String())
			return j.String()
		}
	}

	return ""
}

func runNmap(lnet string, osdisc bool) []byte {
	nmapExec := "/usr/bin/nmap"
	nmapOpts := []string{
    "--system-dns",
    "-oX", "-",
		"-p", "22,23,25,80,123,161,162,443"}
  if osdisc {
    nmapOpts = append( nmapOpts, "-O")
  }
  nmapOpts = append( nmapOpts, lnet)
	log.Printf("Running %s %s", nmapExec, strings.Join(nmapOpts, " "))
	e, out := runExternal(nmapExec, nmapOpts...)
	log.Printf("Done Exit: %d", e)
	log.Printf("Output length %d characters", len(out))
	if e != 0 {
		log.Fatal("Exit %d when running %s", e, nmapExec)
	}
	return []byte(out)
}

func main() {
	var err error
	xmlblob := []byte{}

	// print license
	license.LogLicense()

	// Do the CLI flags
	scanfile := flag.String("xml", "", "XML Output from nmap")
	gname := flag.String("group", "Internal-LAN",
		"Group for hosts on this network")
  outfile := flag.String("output", "raven", "Output filename without exetention")
	lnet := flag.String("net", "", "CIDR network to scan with nmap")
	dhcpRange := flag.String("dhcp", "100-200", "DHCP address range")
	iniFile := flag.Bool("ini", false, "Output to file (ini format)")
	jsonFile := flag.Bool("json", false, "Output to file (json format)")
	baseini := flag.String("skel", "base.ini", "Skeleton file for ini file generaton")
	disabled := flag.Bool("disabled", false, "Mark all hosts as enabled=false")
  osDiscovery := flag.Bool( "osdiscovery", false, "Run Host Discovery (Requires Admin)")
	flag.Parse()

	groupName := *gname
  if _,err := os.Stat(*baseini); err != nil && *iniFile {
    log.Fatal(err)
  }

	// if scanfile is empty try running nmap
	if *scanfile == "" {
		localNet := *lnet
		if localNet == "" {
			log.Printf("Detecting interface")
			localNet = findLocal()
		}
		// give up, no local net found and none provided
		if localNet == "" {
			log.Fatal("no interfaces found")
		}
		xmlblob = runNmap(localNet,*osDiscovery)
	} else {
		// read a provided XML file
		log.Printf("Starting reading %s", scanfile)
		xmlblob, err = ioutil.ReadFile(*scanfile)
		if err != nil {
			log.Fatal(err)
		}
	}

	// We should have an xmlblob by now
	if err := xml.Unmarshal(xmlblob, &nmap); err != nil {
		log.Fatal(err)
	}

//  if _,filename,_,ok := runtime.Caller(0); ok {
//    log.Printf("filepath: %s\n", path.Join(path.Dir(filename)))
//  }
  cfg := ini.Empty()
  main := cfg.Section("")
  main.Comment = fmt.Sprintf("%s %s\nRun Started at %s", nmap.Scanner, nmap.Args, nmap.StartS)
  if *iniFile {
    if _,err := os.Stat(*baseini); err != nil {
      log.Fatal(err)
    }
	  err := cfg.Append(*baseini)
    if err != nil {
      log.Fatal(err)
    }
  }
	hosts := make(map[string]*HostJSON)
	portsEnabled := make(map[int][]string)

	// This may need to get better
	dhcp := strings.Split(*dhcpRange, "-")
	dhcplow, _ := strconv.Atoi(dhcp[0])
	dhcphi, _ := strconv.Atoi(dhcp[1])
	log.Printf("DHCP range is %d to %d", dhcplow, dhcphi)

	for _, v := range nmap.Host {
		hr := new(HostJSON)
		hr.Name, hr.Hostname, hr.IPv4, hr.DHCP = getHostInfo(v, dhcplow, dhcphi)
    hr.OS = v.OSInfo.Name
		hr.When = time.Unix(v.EndTime, 0)
		hr.Enabled = !*disabled
		section,_ := cfg.NewSection(hr.Name)
    if hr.Hostname != "" {
      section.Key("hostname").SetValue(hr.Hostname)
    }
		section.Key("group").SetValue(groupName)
		section.Key("enabled").SetValue(fmt.Sprintf("%t", hr.Enabled))
		if !hr.DHCP {
			section.Key("ipv4").SetValue(hr.IPv4)
		}
		portsEnabled[0] = append(portsEnabled[0], hr.Name)
		for _, p := range v.Ports {
			if p.State.State == "open" {
				hr.Ports = append(hr.Ports, p.PortID)
				portsEnabled[p.PortID] = append(portsEnabled[p.PortID], hr.Name)
			}
		}
    section.Comment = fmt.Sprintf("Host: %s (%s)\nOpen Ports: %v", hr.Hostname, hr.IPv4, hr.Ports)
		log.Printf("%s(%s) %v", hr.Name, hr.IPv4, hr.Ports)
		hosts[hr.Name] = hr
	}
  var keys []int
  for k:= range portsEnabled {
    keys = append(keys, k)
  }
  sort.Ints(keys)
	for _,k := range keys {
    v := portsEnabled[k]
		log.Printf("Port:%d %s", k, strings.Join(v, " "))
		c := []string{"Check", strconv.Itoa(k)}
		section,err := cfg.GetSection(strings.Join(c, "-"))
    if err != nil {
      continue
    }
		section.Key("hosts").SetValue(strings.Join(v, " "))
	}

	if *iniFile {
		var b strings.Builder
		b.WriteString(*outfile)
		b.WriteString(".ini")
		outfile := b.String()

		log.Printf("Writing %s", outfile)
		cfg.SaveToIndent(outfile, "  ")
	}

	if *jsonFile {
		var b strings.Builder
		b.WriteString(*outfile)
		b.WriteString(".json")
		outfile := b.String()

		l, _ := json.MarshalIndent(&hosts, "", "  ")
		log.Printf("Writing %s", outfile)
		ioutil.WriteFile(outfile, l, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("Done...")
}
