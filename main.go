package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const banner = `#        WaybackScope URL Collector v1.0.0        #
#        Developed by @h6nt3r                      #

[!] Legal disclaimer: Usage of WaybackScope for attacking targets without prior mutual
    consent is illegal. It is the end user's responsibility to obey all applicable
    local, state and federal laws. Developers assume no liability and are not
    responsible for any misuse or damage caused by this program.

`

func printBanner() {
	fmt.Print(banner)
}

func hasHelpFlag() bool {
	for _, a := range os.Args[1:] {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

// normalizeDomain tries to convert various input formats into a clean host name.
// Examples:
//  - "https://example.com/" -> "example.com"
//  - "example.com/"         -> "example.com"
//  - "sub.example.com"      -> "sub.example.com"
func normalizeDomain(line string) string {
	d := strings.TrimSpace(line)
	if d == "" {
		return ""
	}

	// If it looks like a URL with scheme, parse it.
	if strings.HasPrefix(d, "http://") || strings.HasPrefix(d, "https://") {
		u, err := url.Parse(d)
		if err == nil && u.Host != "" {
			d = u.Host
		} else {
			// fallback: strip scheme manually
			d = strings.TrimPrefix(d, "http://")
			d = strings.TrimPrefix(d, "https://")
		}
	}

	// Strip any trailing slashes
	for strings.HasSuffix(d, "/") {
		d = strings.TrimSuffix(d, "/")
	}

	return d
}

func pipedInputDomains() ([]string, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	// if there's data piped in (not a terminal)
	if (info.Mode() & os.ModeCharDevice) == 0 {
		var ds []string
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			norm := normalizeDomain(line)
			if norm != "" {
				ds = append(ds, norm)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return ds, nil
	}
	return nil, nil
}

var timeoutErrors int64
var otherErrors int64

type targetDomain struct {
	Name  string
	Exact bool
}

func main() {
	domain := flag.String("d", "", "Target domain with subdomains (e.g., example.com)")
	domainExact := flag.String("u", "", "Target domain only (no subdomains, e.g., example.com)")
	domainList := flag.String("dl", "", "File containing list of domains (one per line)")
	outputFile := flag.String("o", "", "Output file (e.g., result.txt)")
	timeout := flag.Int("t", 10, "Timeout in seconds")
	workers := flag.Int("w", 5, "Number of concurrent workers (default 5)")
	silentFlag := flag.Bool("s", false, "Silent mode: terminal prints ONLY URLs (no banner, no summary, no other messages)")
	userAgent := flag.String("ua", "WaybackScope/1.0 (@h6nt3r)", "Custom User-Agent for Wayback requests")
	retries := flag.Int("retries", 2, "Number of retries per domain on transient errors")
	delayMs := flag.Int("delay-ms", 0, "Delay in milliseconds between processing domains")

	// Custom Usage: ordered & readable
	flag.Usage = func() {
		printBanner()
		fmt.Fprintln(flag.CommandLine.Output(), "-u\tTarget domain only (no subdomains, e.g., example.com)")
		fmt.Fprintln(flag.CommandLine.Output(), "-d\tTarget domain with subdomains (e.g., example.com)")
		fmt.Fprintln(flag.CommandLine.Output(), "-dl\tFile containing list of domains (one per line)")
		fmt.Fprintln(flag.CommandLine.Output(), "-t\tTimeout in seconds")
		fmt.Fprintln(flag.CommandLine.Output(), "-w\tNumber of concurrent workers (default 5)")
		fmt.Fprintln(flag.CommandLine.Output(), "-s\tSilent mode: terminal prints ONLY URLs (no banner, no summary, no other messages)")
		fmt.Fprintln(flag.CommandLine.Output(), "-o\tOutput file (e.g., result.txt)")
		fmt.Fprintln(flag.CommandLine.Output(), "-ua\tCustom User-Agent for Wayback requests")
		fmt.Fprintln(flag.CommandLine.Output(), "-retries\tNumber of retries per domain on transient errors")
		fmt.Fprintln(flag.CommandLine.Output(), "-delay-ms\tDelay in milliseconds between processing domains")
		fmt.Fprintln(flag.CommandLine.Output(), "")
	}

	flag.Parse()

	if hasHelpFlag() {
		flag.Usage()
		return
	}

	silent := *silentFlag

	if !silent {
		printBanner()
	}

	// Prepare list of target domains (with Exact flag)
	var targets []targetDomain

	// 1) piped input, if any, has highest precedence (treated as Exact=true)
	stdinDomains, _ := pipedInputDomains()
	if len(stdinDomains) > 0 {
		for _, d := range stdinDomains {
			targets = append(targets, targetDomain{
				Name:  d,
				Exact: true,
			})
		}
	} else {
		// 2) domain list file
		if *domainList != "" {
			f, err := os.Open(*domainList)
			if err != nil {
				if !silent {
					fmt.Printf("Error reading domain list: %v\n", err)
				}
				return
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				norm := normalizeDomain(line)
				if norm == "" {
					continue
				}
				// by default, treat list domains as wildcard (subdomains included)
				targets = append(targets, targetDomain{
					Name:  norm,
					Exact: false,
				})
			}
			if err := scanner.Err(); err != nil {
				if !silent {
					fmt.Printf("Error reading domain list: %v\n", err)
				}
				return
			}
		} else {
			// 3) single domain flags
			if *domain != "" {
				norm := normalizeDomain(*domain)
				if norm != "" {
					targets = append(targets, targetDomain{
						Name:  norm,
						Exact: false,
					})
				}
			}
			if *domainExact != "" {
				norm := normalizeDomain(*domainExact)
				if norm != "" {
					targets = append(targets, targetDomain{
						Name:  norm,
						Exact: true,
					})
				}
			}
		}
	}

	if len(targets) == 0 {
		if !silent {
			flag.Usage()
		}
		return
	}

	// Prepare output file writer (streaming)
	var fileWriter *bufio.Writer
	var outFile *os.File
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			if !silent {
				fmt.Printf("Error creating file: %v\n", err)
			}
			return
		}
		outFile = f
		fileWriter = bufio.NewWriter(outFile)
	}

	// Tuned transport for better speed
	tr := &http.Transport{
		MaxIdleConns:          500,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(*timeout) * time.Second,
	}

	startTime := time.Now()

	targetChan := make(chan targetDomain, len(targets))
	urlChan := make(chan string, 1000)
	var wg sync.WaitGroup

	// workers
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for td := range targetChan {
				streamURLs(td, client, urlChan, *userAgent, *retries, silent)
				// Delay between domains, if configured
				if *delayMs > 0 {
					time.Sleep(time.Duration(*delayMs) * time.Millisecond)
				}
			}
		}()
	}

	// feed targets
	go func() {
		for _, td := range targets {
			targetChan <- td
		}
		close(targetChan)
	}()

	// close urlChan when done
	go func() {
		wg.Wait()
		close(urlChan)
	}()

	// streaming output: stdout + optional file
	var total int
	for u := range urlChan {
		// Always print URLs, even in silent mode
		fmt.Println(u)
		if fileWriter != nil {
			fileWriter.WriteString(u + "\n")
		}
		total++
	}

	if fileWriter != nil {
		fileWriter.Flush()
		outFile.Close()
		if !silent {
			fmt.Printf("[+] Saved %d URLs to %s\n", total, *outputFile)
		}
	}

	elapsed := time.Since(startTime)
	minutes := int(elapsed.Minutes())
	seconds := int(elapsed.Seconds()) - minutes*60
	if !silent {
		fmt.Printf("Time taken: %d Minute %d Second\n", minutes, seconds)
		fmt.Printf("Timeout Errors: %d\n", atomic.LoadInt64(&timeoutErrors))
		fmt.Printf("Other Errors:   %d\n", atomic.LoadInt64(&otherErrors))
	}
}

func streamURLs(td targetDomain, client *http.Client, out chan<- string, ua string, retries int, silent bool) {
	domain := td.Name
	exact := td.Exact

	var apiURL string
	if exact {
		// Only the exact domain
		apiURL = fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=%s/*&collapse=urlkey&output=text&fl=original", domain)
	} else {
		// Include subdomains
		apiURL = fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=*.%s/*&collapse=urlkey&output=text&fl=original", domain)
	}

	try := 0
	for {
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			atomic.AddInt64(&otherErrors, 1)
			return
		}
		req.Header.Set("User-Agent", ua)

		resp, err := client.Do(req)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				atomic.AddInt64(&timeoutErrors, 1)
			} else {
				atomic.AddInt64(&otherErrors, 1)
			}

			try++
			if try <= retries {
				// simple backoff
				time.Sleep(time.Duration(500*try) * time.Millisecond)
				continue
			}
			return
		}

		func() {
			defer resp.Body.Close()

			reader := bufio.NewReader(resp.Body)
			for {
				line, err := reader.ReadString('\n')
				if err == io.EOF {
					line = strings.TrimSpace(line)
					if line != "" {
						out <- line
					}
					break
				}
				if err != nil {
					atomic.AddInt64(&otherErrors, 1)
					break
				}
				urlStr := strings.TrimSpace(line)
				if urlStr != "" {
					out <- urlStr
				}
			}
		}()

		// Successfully processed response; no more retries
		return
	}
}
