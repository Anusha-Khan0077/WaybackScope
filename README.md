
# **WaybackScope**

WaybackScope is a high-speed **Wayback Machine URL harvesting engine** built for security researchers, bug bounty hunters, and reconnaissance workflows.

It pulls historical URLs directly from the **Internet Archive CDX API**, supports wildcard subdomain expansion, piped input, concurrency, retries, custom User-Agent, silent streaming mode, and clean domain normalization.

> ⚠ **Legal Notice:** Only use WaybackScope on targets you are authorized to test.
> Misuse can violate laws. The developer holds zero liability for damage or misuse.

---

## 🚀 **Features**

### ✔ High-speed URL extraction

Queries Wayback’s CDX API with optimized concurrency and HTTP transport tuning.

### ✔ Domain modes

* **Exact domain**

  ```
  ./waybackscope -u example.com
  → url=example.com/*
  ```
* **Wildcard domain**

  ```
  ./waybackscope -d example.com
  → url=*.example.com/*
  ```
* **Domain list file** (`-dl`)
* **Piped input** (treated as exact domains)

### ✔ Fully silent streaming mode

`-s` prints **only URLs** — ideal for chaining into pipelines:

```
./waybackscope -d example.com -s | httpx -silent | nuclei -silent
```

### ✔ Resilient request engine

* Timeout handling
* Retries (`-retries`)
* Delay between domains (`-delay-ms`)

### ✔ Custom User-Agent

```
-ua "ReconTool/2.0"
```

### ✔ Domain normalization

Accepts messy input like:

* `https://example.com/`
* `http://sub.example.com/login`
* `example.com/`
* `sub.example.com/path/`

All converted to clean hostnames.

---

## 🔧 **Requirements**

* Go 1.18+
* Internet access (Archive.org)

No external libraries — **pure Go standard library**.

---

## 📦 **Installation**

```bash
git clone https://github.com/YOUR_USERNAME/WaybackScope.git
cd WaybackScope
go build -o waybackscope main.go
```

---

## ▶ **Usage**

```
-u          Target domain only (no subdomains)
-d          Target domain with subdomains
-dl         File containing list of domains
-t          Timeout in seconds
-w          Number of workers (default 5)
-s          Silent mode (URLs only)
-o          Output file (e.g., result.txt)
-ua         Custom User-Agent
-retries    Retries on transient errors
-delay-ms   Delay between domain requests (ms)
```

---

## 🧪 **Examples**

### Single domain (wildcard)

```bash
./waybackscope -d example.com
```

### Exact domain

```bash
./waybackscope -u example.com
```

### Domain list (wildcard mode)

```bash
./waybackscope -dl domains.txt -w 10
```

### Silent pipeline

```bash
./waybackscope -d example.com -s | anew
```

### Custom UA + retries

```bash
./waybackscope -d example.com -ua "MyAgent/1.0" -retries 3
```

### Piped input

```bash
cat roots.txt | ./waybackscope -s
```

---

## 📊 **Output Summary (non-silent mode)**

Shows:

* Total time
* Timeout errors
* Other errors
* Optional output file path

In silent mode **none of this appears** — only URLs.

---

## ⚖ **Ethical / Legal**

WaybackScope is intended for:

* Reconnaissance
* Security assessments
* Bug bounty research

Use ONLY on systems where you have permission.
Unauthorized scraping or probing can violate local laws.

