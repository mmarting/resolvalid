package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

var version = "2.0.0"

var (
	file       string
	url        string
	output     string
	testDomain string
	threads    int
	quiet     bool
	help       bool
	showVer    bool
	timeout    time.Duration
	maxLatency time.Duration
	retries    int
)

var defaultTestDomains = []string{
	"resolvalid.mmartin.me",
	"resolvalid2.mmartin.me",
	"resolvalid3.mmartin.me",
}

var publicDNSServers = []string{
	"1.1.1.1",
	"8.8.8.8",
	"8.8.4.4",
}

var defaultDNSListURL = "https://public-dns.info/nameservers.txt"

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

var useColor bool

func init() {
	flag.StringVar(&file, "file", "", "File containing the list of DNS servers (optional)")
	flag.StringVar(&file, "f", "", "File containing the list of DNS servers (shorthand, optional)")

	flag.StringVar(&url, "url", "", "URL containing the file of DNS servers (optional)")
	flag.StringVar(&url, "u", "", "URL containing the file of DNS servers (shorthand, optional)")

	flag.StringVar(&output, "output", "", "Output file for valid DNS servers (required)")
	flag.StringVar(&output, "o", "", "Output file for valid DNS servers (shorthand, required)")

	flag.StringVar(&testDomain, "test-domain", "", "Domain used to test DNS servers")
	flag.StringVar(&testDomain, "td", "", "Domain used to test DNS servers (shorthand)")

	flag.IntVar(&threads, "threads", 20, "Number of concurrent threads (default 20)")
	flag.IntVar(&threads, "t", 20, "Number of concurrent threads (shorthand, default 20)")

	flag.DurationVar(&timeout, "timeout", 2*time.Second, "Timeout for DNS queries (default 2s)")
	flag.DurationVar(&timeout, "to", 2*time.Second, "Timeout for DNS queries (shorthand, default 2s)")

	flag.DurationVar(&maxLatency, "max-latency", 0, "Maximum acceptable response time (optional, e.g. 500ms)")
	flag.DurationVar(&maxLatency, "ml", 0, "Maximum acceptable response time (shorthand, optional)")

	flag.IntVar(&retries, "retries", 0, "Number of retries for failed DNS queries (default 0)")
	flag.IntVar(&retries, "r", 0, "Number of retries for failed DNS queries (shorthand, default 0)")

	flag.BoolVar(&quiet, "quiet", false, "Suppress output to the screen")
	flag.BoolVar(&quiet, "q", false, "Suppress output to the screen (shorthand)")

	flag.BoolVar(&showVer, "version", false, "Display version information")
	flag.BoolVar(&showVer, "v", false, "Display version information (shorthand)")

	flag.BoolVar(&help, "help", false, "Display help information")
	flag.BoolVar(&help, "h", false, "Display help information (shorthand)")

	flag.Usage = func() {
		printLogo()
		printUsage()
	}
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	flag.Parse()

	useColor = isTerminal(os.Stdout) && !quiet

	if showVer {
		fmt.Printf("resolvalid v%s\n", version)
		return
	}

	if help || (output == "") {
		printLogo()
		printUsage()
		return
	}

	stdinPiped := !isTerminal(os.Stdin)

	if file == "" && url == "" && stdinPiped {
		fmt.Println("Reading DNS servers from stdin.")
	} else if file == "" && url == "" {
		url = defaultDNSListURL
		fmt.Println("No DNS server source file or URL provided. Using default public DNS list.")
	}

	if testDomain == "" {
		testDomain = defaultTestDomains[rand.Intn(len(defaultTestDomains))]
	}

	expectedIPs, err := getExpectedIPs(testDomain)
	if err != nil {
		fmt.Printf("Failed to get expected IPs: %v\n", err)
		return
	}

	dnsServers, err := getDNSServers()
	if err != nil {
		fmt.Printf("Failed to get DNS servers: %v\n", err)
		return
	}

	dnsServers = filterValidIPs(dnsServers)
	if len(dnsServers) == 0 {
		fmt.Println("Error: No valid IP addresses found in the provided DNS server list.")
		return
	}

	checkDNSServers(dnsServers, expectedIPs)
}

func printLogo() {
	fmt.Print(`
 ____                 _             _ _     _
|  _ \ ___  ___  ___ | |_   ____ _| (_) __| |
| |_) / _ \/ __|/ _ \| \ \ / / _` + "`" + ` | | |/ _` + "`" + ` |
|  _ <  __/\__ \ (_) | |\ V / (_| | | | (_| |
|_| \_\___||___/\___/|_| \_/ \__,_|_|_|\__,_|
`)
}

func printUsage() {
	fmt.Printf("  resolvalid v%s\n", version)
	fmt.Println("\nAuthor:")
	fmt.Println("  Name:               Mart\u00edn Mart\u00edn")
	fmt.Println("  Website:            https://mmartin.me/")
	fmt.Println("  LinkedIn:           https://www.linkedin.com/in/martinmarting/")
	fmt.Println("  GitHub:             https://github.com/mmarting/resolvalid")

	fmt.Println("\nUsage:")
	fmt.Println("  resolvalid -o <output_file> [options]")
	fmt.Println("  cat servers.txt | resolvalid -o <output_file>")

	fmt.Println("\nOptions:")
	fmt.Println("  -o, --output        Output file for valid DNS servers (required)")
	fmt.Println("  -f, --file          File containing the list of DNS servers (optional)")
	fmt.Println("  -u, --url           URL containing the file of DNS servers (optional, default: https://public-dns.info/nameservers.txt)")
	fmt.Println("  -td, --test-domain  Domain used to test DNS servers (optional, default: randomly chosen from predefined domains)")
	fmt.Println("  -t, --threads       Number of concurrent threads (optional, default: 20)")
	fmt.Println("  -to, --timeout      Timeout for DNS queries (optional, default: 2s)")
	fmt.Println("  -ml, --max-latency  Maximum acceptable response time (optional, e.g. 500ms)")
	fmt.Println("  -r, --retries       Number of retries for failed DNS queries (optional, default: 0)")
	fmt.Println("  -q, --quiet         Suppress output to the screen (optional)")
	fmt.Println("  -v, --version       Display version information")
	fmt.Println("  -h, --help          Display help information")

	fmt.Println("\nExamples:")
	fmt.Println("  1. Use a local file with DNS servers and output valid ones to a file:")
	fmt.Println("     resolvalid -f dns_servers.txt -o valid_servers.txt")
	fmt.Println("\n  2. Use a URL for DNS servers and output valid ones to a file with custom timeout:")
	fmt.Println("     resolvalid -u https://example.com/dns_list.txt -o valid_servers.txt -to 5s")
	fmt.Println("\n  3. Suppress screen output and use custom test domain:")
	fmt.Println("     resolvalid -f dns_servers.txt -o valid_servers.txt -td mytestdomain.com -q")
	fmt.Println("\n  4. Pipe DNS servers from stdin:")
	fmt.Println("     cat dns_servers.txt | resolvalid -o valid_servers.txt")
	fmt.Println("\n  5. Only keep servers that respond within 500ms with 2 retries:")
	fmt.Println("     resolvalid -f dns_servers.txt -o valid_servers.txt -ml 500ms -r 2")
}

func filterValidIPs(servers []string) []string {
	var valid []string
	skipped := 0
	for _, s := range servers {
		if net.ParseIP(s) != nil {
			valid = append(valid, s)
		} else {
			skipped++
		}
	}
	if skipped > 0 && !quiet {
		fmt.Printf("%sSkipped %d invalid IP address(es) from the input list.%s\n", colorYellow, skipped, colorReset)
	}
	return valid
}

func getExpectedIPs(domain string) ([]string, error) {
	var expectedIPs []string
	for _, publicDNS := range publicDNSServers {
		c := new(dns.Client)
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(domain), dns.TypeA)

		in, _, err := c.Exchange(m, net.JoinHostPort(publicDNS, "53"))
		if err != nil {
			continue
		}

		for _, ans := range in.Answer {
			if a, ok := ans.(*dns.A); ok {
				expectedIPs = append(expectedIPs, a.A.String())
			}
		}
	}

	if len(expectedIPs) == 0 {
		return nil, fmt.Errorf("failed to resolve test domains, please provide new ones using --test-domain or -td option")
	}

	return expectedIPs, nil
}

func getDNSServers() ([]string, error) {
	if file != "" {
		return readDNSServersFromFile(file)
	} else if url != "" {
		return readDNSServersFromURL(url)
	} else if !isTerminal(os.Stdin) {
		return readDNSServersFromStdin()
	}
	return nil, fmt.Errorf("failed to read the source DNS servers list")
}

func readDNSServersFromFile(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var servers []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		server := strings.TrimSpace(scanner.Text())
		if server != "" {
			servers = append(servers, server)
		}
	}

	return servers, scanner.Err()
}

func readDNSServersFromURL(fileURL string) ([]string, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var servers []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		server := strings.TrimSpace(scanner.Text())
		if server != "" {
			servers = append(servers, server)
		}
	}

	return servers, scanner.Err()
}

func readDNSServersFromStdin() ([]string, error) {
	var servers []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		server := strings.TrimSpace(scanner.Text())
		if server != "" {
			servers = append(servers, server)
		}
	}

	return servers, scanner.Err()
}

func checkDNSServers(servers []string, expectedIPs []string) {
	var mu sync.Mutex
	totalServers := len(servers)
	var checkedServers, validServers, nonValidServers int

	outputFile, err := os.Create(output)
	if err != nil {
		fmt.Printf("Error: Failed to create output file: %v\n", err)
		return
	}
	defer outputFile.Close()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, threads)

	for _, server := range servers {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(server string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			valid, err := checkDNSServerWithRetries(server, expectedIPs)
			mu.Lock()
			if err == nil && valid {
				validServers++
				outputFile.WriteString(server + "\n")
			} else {
				nonValidServers++
			}
			checkedServers++
			completion := float64(checkedServers) / float64(totalServers) * 100
			if !quiet {
				validColor := colorGreen
				nonValidColor := colorRed
				if !useColor {
					validColor = ""
					nonValidColor = ""
				}
				reset := colorReset
				if !useColor {
					reset = ""
				}
				fmt.Printf("\rChecking %d of %d DNS servers. Results: %s%d valid%s - %s%d non-valid%s DNS servers. %.2f%% completed.",
					checkedServers, totalServers,
					validColor, validServers, reset,
					nonValidColor, nonValidServers, reset,
					completion)
			}
			mu.Unlock()
		}(server)
	}

	wg.Wait()
	if !quiet {
		fmt.Println()
	}
}

func checkDNSServerWithRetries(server string, expectedIPs []string) (bool, error) {
	var valid bool
	var err error

	for attempt := 0; attempt <= retries; attempt++ {
		valid, err = checkSingleDNSServer(server, expectedIPs)
		if err == nil && valid {
			return true, nil
		}
	}

	return valid, err
}

func checkSingleDNSServer(server string, expectedIPs []string) (bool, error) {
	c := &dns.Client{
		Timeout: timeout,
	}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	in, rtt, err := c.Exchange(m, net.JoinHostPort(server, "53"))
	if err != nil {
		return false, err
	}

	if maxLatency > 0 && rtt > maxLatency {
		return false, nil
	}

	if len(in.Answer) == 0 {
		return false, nil
	}

	for _, ans := range in.Answer {
		if a, ok := ans.(*dns.A); ok {
			for _, expectedIP := range expectedIPs {
				if a.A.String() == expectedIP {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
