# Resolvalid

Resolvalid is a Go tool that will generate or validate a list of valid DNS servers.

## Installation

To install Resolvalid, use the `go install` command:

```sh
go install github.com/mmarting/resolvalid@latest
```

## Usage

Use -h to display the help for the tool:

```sh
resolvalid -h
```

Resolvalid requires an output (-o) file name as the only mandatory parameter. The tool admits the following options:

## Options

    -o, --output:       Output file for valid DNS servers (required).
    -f, --file:         File containing the list of DNS servers (optional).
    -u, --url:          URL containing the file of DNS servers (optional, default: https://public-dns.info/nameservers.txt).
    -td, --test-domain: Domain used to test DNS servers (optional, default: randomly chosen from predefined domains).
    -t, --threads:      Number of concurrent threads (optional, default: 20).
    -to, --timeout:     Timeout for DNS queries (optional, default: 2s).
    -s, --silent :      Suppress output to the screen (optional).
    -h, --help:         Display help information.

## Examples

Use a local file with DNS Servers and output valid ones to a file:

```sh
resolvalid -f dns_servers.txt -o valid_servers.txt
```

Use a URL hosting a list of DNS Servers and output valid ones to a file with custom timeout:

```sh
resolvalid -u https://example.com/dns_list.txt -o valid_servers.txt -to 5s
```

Suppress screen output, use custom test domain and use 50 threads:

```sh
resolvalid -f dns_servers.txt -o valid_servers.txt -td mytestdomain.com -t 50 -s
```

## Notes

This tool was created as a project for learning Go, so don't expect a highly optimized code. At the same time, I was trying to reduce the time my recon automation tool was taking to validate my DNS servers list, which I've achieved with this tool.

## Thanks

- vortexau: For creating the original tool I've been using for the last few years to validate my DNS server lists, which inspired me to build a similar and more optimized tool in Go: [dnsvalidator](https://github.com/vortexau/dnsvalidator)
- miek: For creating the Go DNS library I'm using in this tool: [dns](https://github.com/vortexau/dnsvalidator)

## License

`resolvalid` is distributed under GPL v3 License.
