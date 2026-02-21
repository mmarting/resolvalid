# Resolvalid

[![Go Version](https://img.shields.io/github/go-mod/go-version/mmarting/resolvalid)](https://go.dev/)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

Resolvalid is a fast, concurrent DNS server validator written in Go. Given a list of DNS servers, it tests each one and outputs only the servers that return correct results — giving you a clean, reliable resolver list for your workflows.

**Current version: 2.0.0**

## Table of Contents

- [How It Works](#how-it-works)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Building from Source](#building-from-source)
- [Usage](#usage)
- [Options](#options)
- [Examples](#examples)
- [Thanks](#thanks)
- [Author](#author)
- [License](#license)

## How It Works

1. Resolvalid resolves a **test domain** against trusted public DNS servers (Cloudflare `1.1.1.1`, Google `8.8.8.8` / `8.8.4.4`) to establish the **expected IP addresses**.
2. It then queries each DNS server in your list with the same test domain, using concurrent goroutines for speed.
3. A server is considered **valid** only if it returns one of the expected IP addresses — meaning it resolves correctly and isn't hijacking or dropping queries.
4. Valid servers are written to the output file, one per line.

This approach detects DNS servers that are down, misconfigured, censoring results, or returning poisoned responses.

## Prerequisites

- **Go 1.24+** (for installation via `go install` or building from source)

## Installation

```sh
go install github.com/mmarting/resolvalid@latest
```

## Building from Source

```sh
git clone https://github.com/mmarting/resolvalid.git
cd resolvalid
go build -o resolvalid .
```

## Usage

```sh
resolvalid -o <output_file> [options]
```

If no input source is provided (`-f` or `-u`), resolvalid will read from **stdin** if piped, or fall back to a default public DNS list from [public-dns.info](https://public-dns.info/nameservers.txt).

Use `-h` to display all available options:

```sh
resolvalid -h
```

## Options

| Flag | Long Flag | Description | Default |
|------|-----------|-------------|---------|
| `-o` | `--output` | Output file for valid DNS servers | **(required)** |
| `-f` | `--file` | File containing the list of DNS servers | — |
| `-u` | `--url` | URL containing the file of DNS servers | `https://public-dns.info/nameservers.txt` |
| `-td` | `--test-domain` | Domain used to test DNS servers | Randomly chosen from predefined domains |
| `-t` | `--threads` | Number of concurrent threads | `20` |
| `-to` | `--timeout` | Timeout for DNS queries | `2s` |
| `-ml` | `--max-latency` | Maximum acceptable response time | — (disabled) |
| `-r` | `--retries` | Number of retries for failed DNS queries | `0` |
| `-q` | `--quiet` | Suppress output to the screen | `false` |
| `-v` | `--version` | Display version information | — |
| `-h` | `--help` | Display help information | — |

## Examples

**Basic usage** — validate using the default public DNS list:

```sh
resolvalid -o valid_servers.txt
```

**Use a local file** with DNS servers:

```sh
resolvalid -f dns_servers.txt -o valid_servers.txt
```

**Use a URL** hosting a list of DNS servers with a custom timeout:

```sh
resolvalid -u https://example.com/dns_list.txt -o valid_servers.txt -to 5s
```

**Pipe from stdin** — composable with other tools:

```sh
cat dns_servers.txt | resolvalid -o valid_servers.txt
```

**Filter by latency** — only keep servers that respond within 500ms, with 2 retries:

```sh
resolvalid -f dns_servers.txt -o valid_servers.txt -ml 500ms -r 2
```

**Quiet mode** with a custom test domain and 50 threads:

```sh
resolvalid -f dns_servers.txt -o valid_servers.txt -td mytestdomain.com -t 50 -q
```

## Thanks

- **vortexau** — For inspiring me to write this tool after years of using theirs to validate DNS server lists: [dnsvalidator](https://github.com/vortexau/dnsvalidator)
- **miek** — For creating the Go DNS library used in this tool: [dns](https://github.com/miekg/dns)

## Author

**Martín Martín**

- [Website](https://mmartin.me/)
- [LinkedIn](https://www.linkedin.com/in/martinmarting/)
- [GitHub](https://github.com/mmarting/resolvalid)

## License

`resolvalid` is distributed under the [GPL v3 License](LICENSE.md).
