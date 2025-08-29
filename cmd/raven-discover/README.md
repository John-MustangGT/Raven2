# Raven Network Discovery Utility

The `raven-discover` utility automatically discovers hosts on your network using nmap and generates a complete Raven configuration file with appropriate checks.

## Features

- Automatic network discovery using nmap
- Generates ping checks for all discovered hosts
- Creates service-specific checks based on open ports
- Handles DHCP vs static IP configuration
- OS detection support (when run as root)
- YAML configuration output compatible with Raven v2

## Usage

### Basic Usage

```bash
# Auto-detect local network and scan
./bin/raven-discover

# Scan specific network
./bin/raven-discover -network 192.168.1.0/24

# Use existing nmap XML file
./bin/raven-discover -xml scan-results.xml
```

### Advanced Options

```bash
# Full example with all options
./bin/raven-discover \
  -network 192.168.1.0/24 \
  -output my-config.yaml \
  -group "production" \
  -dhcp "100-199" \
  -os \
  -enabled=true \
  -verbose
```

### Command Line Options

- `-network <CIDR>`: Network to scan (e.g., 192.168.1.0/24). Auto-detected if not specified.
- `-xml <file>`: Use existing nmap XML file instead of scanning
- `-output <file>`: Output configuration file (default: config.yaml)
- `-group <name>`: Group name for discovered hosts (default: "discovered")
- `-dhcp <range>`: DHCP range like "100-200" (hosts in range won't get static IP)
- `-nmap <path>`: Path to nmap binary (default: /usr/bin/nmap)
- `-enabled`: Mark discovered hosts as enabled (default: true)
- `-os`: Enable OS detection (requires root privileges)
- `-verbose`: Verbose nmap output

## Generated Checks

The utility automatically creates checks based on discovered services:

### Port-Based Checks

- **Port 22 (SSH)**: Uses check_ssh plugin
- **Port 23 (Telnet)**: Uses check_tcp plugin
- **Port 25 (SMTP)**: Uses check_smtp plugin
- **Port 80 (HTTP)**: Uses check_http plugin
- **Port 123 (NTP)**: Uses check_ntp plugin
- **Port 161 (SNMP)**: Uses check_snmp plugin with public community
- **Port 162 (SNMP Trap)**: Uses check_tcp with UDP
- **Port 443 (HTTPS)**: Uses check_http with SSL and certificate checking

### Universal Checks

- **Ping Check**: Applied to all discovered hosts with adaptive intervals

## Configuration Structure

The generated configuration includes:

```yaml
server:
  port: ":8000"
  workers: 3
  # ... other server settings

hosts:
  - id: "server1"
    name: "server1"
    display_name: "server1"
    ipv4: "192.168.1.10"    # Only if not in DHCP range
    hostname: "server1.local"
    group: "discovered"
    enabled: true
    tags:
      os: "Linux 3.2 - 4.9"
      open_ports: "22,80,443"
      discovered: "2024-01-15T10:30:00Z"

checks:
  - id: "ping-check"
    name: "Ping Check"
    type: "ping"
    hosts: ["server1", "server2", "..."]
    # ... check configuration
```

## DHCP Range Handling

Hosts with IPs in the DHCP range will:
- Not have `ipv4` field set (rely on hostname resolution)
- Be tagged as potentially dynamic
- Still be monitored normally

## Prerequisites

1. **nmap installed**: The utility requires nmap to be available
2. **Network access**: Ability to scan the target network
3. **Root privileges**: Only needed for OS detection (`-os` flag)

## Examples

### Discover Corporate Network

```bash
# Scan corporate network with OS detection
sudo ./bin/raven-discover \
  -network 10.0.0.0/16 \
  -group "corporate" \
  -dhcp "200-250" \
  -os \
  -output corporate-config.yaml
```

### Use Existing Scan

```bash
# First run nmap separately
nmap --system-dns -oX network-scan.xml -p 22,23,25,80,123,161,162,443 192.168.1.0/24

# Then generate config from XML
./bin/raven-discover -xml network-scan.xml -output config.yaml
```

### Lab Environment

```bash
# Quick setup for lab network
./bin/raven-discover \
  -network 192.168.100.0/24 \
  -group "lab" \
  -dhcp "50-100" \
  -output lab-config.yaml
```

## Building

Add the discovery utility to your build process:

```bash
make discover
```

Or build manually:

```bash
CGO_ENABLED=1 go build -o bin/raven-discover ./cmd/raven-discover
```

## Integration

After generating the configuration:

1. Review and customize the generated config
2. Adjust check intervals and thresholds as needed
3. Add any custom checks or modify service checks
4. Start Raven with the generated configuration:

```bash
./bin/raven -config config.yaml
```

The discovery utility provides a solid foundation that you can then customize for your specific monitoring needs.
