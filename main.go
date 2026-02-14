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

var (
	file       string
	url        string
	output     string
	testDomain string
	threads    int
	silent     bool
	help       bool
	timeout    time.Duration
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

	flag.BoolVar(&silent, "silent", false, "Suppress output to the screen")
	flag.BoolVar(&silent, "s", false, "Suppress output to the screen (shorthand)")

	flag.BoolVar(&help, "help", false, "Display help information")
	flag.BoolVar(&help, "h", false, "Display help information (shorthand)")
}

func main() {
	flag.Parse()

	if help || (output == "") {
		printLogo()
		printUsage()
		return
	}

	if file == "" && url == "" {
		url = defaultDNSListURL
		fmt.Println("No DNS server source file or URL provided. Using default public DNS list.")
	}

	if output == "" {
		fmt.Println("Error: You must provide an output file with --output (-o).")
		printLogo()
		printUsage()
		os.Exit(1)
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

	checkDNSServers(dnsServers, expectedIPs)
}

func printLogo() {
	fmt.Println(`
░▒▓███████▓▒░░▒▓████████▓▒░░▒▓███████▓▒░░▒▓██████▓▒░░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░░▒▓██████▓▒░░▒▓█▓▒░      ░▒▓█▓▒░▒▓███████▓▒░  
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░    ░▒▓█▓▒▒▓█▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓███████▓▒░░▒▓██████▓▒░  ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░    ░▒▓█▓▒▒▓█▓▒░░▒▓████████▓▒░▒▓█▓▒░      ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░             ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▓█▓▒░ ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░             ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▓█▓▒░ ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓███████▓▒░ ░▒▓██████▓▒░░▒▓████████▓▒░▒▓██▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓█▓▒░▒▓███████▓▒░  
`)
}

func printUsage() {
	fmt.Println("Author:")
	fmt.Println("  Name:               Martín Martín")
	fmt.Println("  Website:            https://mmartin.me/")
	fmt.Println("  LinkedIn:           https://www.linkedin.com/in/martinmarting/")
	fmt.Println("  GitHub:             https://github.com/mmarting/resolvalid")

	fmt.Println("\nUsage:")
	fmt.Println("  -o, --output        Output file for valid DNS servers (required)")
	fmt.Println("  -f, --file          File containing the list of DNS servers (optional)")
	fmt.Println("  -u, --url           URL containing the file of DNS servers (optional, default: https://public-dns.info/nameservers.txt)")
	fmt.Println("  -td, --test-domain  Domain used to test DNS servers (optional, default: randomly chosen from [resolvalid.mmartin.me, resolvalid2.mmartin.me, resolvalid3.mmartin.me])")
	fmt.Println("  -t, --threads       Number of concurrent threads (optional, default: 20)")
	fmt.Println("  -to, --timeout      Timeout for DNS queries (optional, default: 2s)")
	fmt.Println("  -s, --silent        Suppress output to the screen (optional)")
	fmt.Println("  -h, --help          Display help information")

	fmt.Println("\nExamples:")
	fmt.Println("  1. Use a local file with DNS servers and output valid ones to a file:")
	fmt.Println("     resolvalid -f dns_servers.txt -o valid_servers.txt")
	fmt.Println("\n  2. Use a URL for DNS servers and output valid ones to a file with custom timeout:")
	fmt.Println("     resolvalid -u https://example.com/dns_list.txt -o valid_servers.txt -to 5s")
	fmt.Println("\n  3. Suppress screen output and use custom test domain:")
	fmt.Println("     resolvalid -f dns_servers.txt -o valid_servers.txt -td mytestdomain.com -s")
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
		return nil, fmt.Errorf("Error: Failed to resolve test domains, please, provide new ones using --test-domain or -td option.")
	}

	return expectedIPs, nil
}

func getDNSServers() ([]string, error) {
	if file != "" {
		return readDNSServersFromFile(file)
	} else if url != "" {
		return readDNSServersFromURL(url)
	}
	return nil, fmt.Errorf("Error: Failed to read the source DNS servers list.")
}

func readDNSServersFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var servers []string
	scanner := bufio.NewScanner(file)
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

			valid, err := CheckDNSServer(server, expectedIPs)
			mu.Lock()
			if err == nil && valid {
				validServers++
				outputFile.WriteString(server + "\n")
			} else {
				nonValidServers++
			}
			checkedServers++
			completion := float64(checkedServers) / float64(totalServers) * 100
			if !silent {
				fmt.Printf("\rChecking %d of %d DNS servers. Results: %d valid - %d non-valid DNS servers. %.2f%% completed.",
					checkedServers, totalServers, validServers, nonValidServers, completion)
			}
			mu.Unlock()
		}(server)
	}

	wg.Wait()
	if !silent {
		fmt.Println()
	}
}

func CheckDNSServer(server string, expectedIPs []string) (bool, error) {
	c := &dns.Client{
		Timeout: timeout,
	}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	in, _, err := c.Exchange(m, net.JoinHostPort(server, "53"))
	if err != nil {
		return false, err
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

