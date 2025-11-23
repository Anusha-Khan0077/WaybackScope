# WaybackScope

**WaybackScope** is a fast Wayback Machine URL collector for recon and web security work.

It queries the Internet Archive’s CDX API to pull **historical URLs** for one or more domains, with options for:

- Exact domains (`example.com`)
- Wildcard subdomains (`*.example.com`)
- Domain lists
- Piped input
- Concurrency, timeouts, retries, and custom User-Agent
- Silent streaming mode suitable for chaining into other tools

> ⚠ Use this tool only on targets you are authorized to test.

---

## ✨ Features

### 🔭 Domain modes

- `-u` – **Exact domain only**

  ```bash
  ./waybackscope -u example.com
  # Queries: url=example.com/*
-d – Domain + subdomains

./waybackscope -d example.com
# Queries: url=*.example.com/*


-dl – Domain list file

./waybackscope -dl domains.txt


One domain per line

Lines starting with # are ignored

Each domain is normalized (schemes/paths stripped)

Piped input – from another tool:

cat roots.txt | ./waybackscope


Piped domains are treated as exact (-u-style).

⚙ Concurrency & robustness

-w – number of workers (concurrent domain fetchers)

-t – HTTP timeout per request (seconds)

-retries – how many times to retry a domain on transient errors

-delay-ms – delay in milliseconds between domains (per worker)

WaybackScope uses a tuned http.Transport and a configurable User-Agent:

-ua – set your own UA (default: WaybackScope/1.0 (@h6nt3r))

🧵 Streaming output

URLs are streamed as they are discovered:

Sent immediately to stdout

Optionally written to a file with -o

Example:

./waybackscope -d example.com -o example_urls.txt


You can pipe directly into other tools:

./waybackscope -dl roots.txt | httpx -silent | nuclei -silent

🤫 Silent mode

Use -s to enable silent mode:

./waybackscope -d example.com -s


In silent mode:

Only URLs are printed

No banner, no stats, no error messages

Ideal for scripting and chaining with other tools

📊 Error tracking

For debugging or tuning (non-silent mode), after execution finishes you’ll see:

Total time taken

Number of timeout errors

Number of other errors (connection, read errors, etc.)

Optional message if results were written to file

This helps you judge whether your timeout/retry settings are too aggressive.

🔧 Requirements

Go (for building)

Internet access (Wayback Machine / archive.org)

No external Go modules beyond the standard library.

🛠 Build & Install
git clone https://github.com/YOUR_USERNAME/WaybackScope.git
cd WaybackScope
go build -o waybackscope main.go


Now you can run:

./waybackscope -h

▶ Usage

Basic flags:

-u          Target domain only (no subdomains, e.g., example.com)
-d          Target domain with subdomains (e.g., example.com)
-dl         File containing list of domains (one per line)
-t          Timeout in seconds
-w          Number of concurrent workers (default 5)
-s          Silent mode: ONLY URLs printed
-o          Output file (e.g., result.txt)
-ua         Custom User-Agent for Wayback requests
-retries    Number of retries per domain on transient errors
-delay-ms   Delay in milliseconds between processing domains


Examples:

# Single domain with subdomains
./waybackscope -d example.com

# Exact domain only
./waybackscope -u example.com

# Domain list (wildcard mode)
./waybackscope -dl domains.txt -w 10 -t 15

# Pipe domains from another tool
cat roots.txt | ./waybackscope -s | tee wayback_urls.txt

# With retries, delay, and custom UA
./waybackscope -d example.com -retries 3 -delay-ms 250 -ua "MyReconTool/1.0"

⚠ Legal / Ethical

This tool is intended for:

Security researchers

Penetration testers

Bug bounty hunters

Use it only on targets where you have explicit permission.
Unauthorized testing or scraping may be illegal in your jurisdiction.
