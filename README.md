# LUMA - Universal Checker

A next-generation high-performance universal account checker that supports multiple configuration formats, advanced parsing engines, and intelligent proxy management. LUMA combines the best features of OpenBullet, SilverBullet, and CyberBullet while introducing innovative enhancements.

## Features

🚀 **Multi-Format Support**
- OpenBullet (.opk) configurations
- SilverBullet (.svb) configurations  
- Loli (.loli) configurations
- OpenBulletPro (.anom) configurations
- Automatic format detection

🧠 **Advanced Parsing Engine**
- JSON field extraction and manipulation
- CSS selector-based HTML parsing
- REGEX pattern matching with capture groups
- Left-Right delimiter parsing
- Modular parsing architecture

🔧 **Enhanced Variable System**
- Dynamic variable types (Single, List, Dictionary)
- Runtime variable management
- Type-safe operations
- Advanced data transformation

🌐 **Proxy Management**
- Automatic proxy scraping (SOCKS4, SOCKS5, HTTP, HTTPS)
- Proxy validation and health checking
- Smart proxy rotation
- Support for custom proxy lists

⚡ **High Performance**
- Multi-threaded worker pool
- Optimized for high CPM (Checks Per Minute)
- Real-time statistics and progress tracking
- Configurable request timeouts and retry logic

📁 **Drag & Drop Support**
- Simply drag config files onto the executable
- Automatic file type detection
- No command line knowledge required

## Quick Start

### 1. Build the Application

```bash
cd universal-checker
go mod tidy
go build -o universal-checker cmd/main.go
```

### 2. Basic Usage

**Command Line:**
```bash
./universal-checker --configs config.opk --combos combos.txt --workers 100
```

**Drag & Drop:**
1. Drag your .opk, .svb, or .loli config files onto the executable
2. Drag your combo file (username:password format)
3. Optionally drag a proxy file
4. The checker will start automatically!

### 3. Advanced Usage

```bash
./universal-checker \
  --configs config1.opk,config2.svb,config3.loli \
  --combos my_combos.txt \
  --proxies my_proxies.txt \
  --workers 200 \
  --output results \
  --request-timeout 30000 \
  --proxy-timeout 5000 \
  --valid-only
```

## Configuration Formats

### OpenBullet (.opk)
```json
{
  "name": "Login Checker",
  "url": "https://example.com/login",
  "method": "POST",
  "headers": {
    "User-Agent": "Mozilla/5.0...",
    "Content-Type": "application/x-www-form-urlencoded"
  },
  "data": {
    "username": "<USER>",
    "password": "<PASS>"
  },
  "conditions": {
    "success": ["welcome", "dashboard"],
    "failure": ["invalid", "error"]
  }
}
```

### SilverBullet (.svb)
```yaml
name: Login Checker
url: https://example.com/login
method: POST
request:
  headers:
    User-Agent: Mozilla/5.0...
  data:
    username: <USER>
    password: <PASS>
response:
  success: ["welcome"]
  failure: ["invalid"]
```

### Loli (.loli)
```
REQUEST POST https://example.com/login
HEADERS User-Agent: Mozilla/5.0...
POSTDATA username=<USER>&password=<PASS>
KEYCHECK Contains "welcome" SUCCESS
KEYCHECK Contains "invalid" FAILURE
CPM 300
```

## File Formats

### Combo Files
```
username:password
user@email.com:password123
admin:secretpass
```

### Proxy Files
```
ip:port
192.168.1.1:8080
proxy.example.com:3128:http
socks5.proxy.com:1080:socks5
```

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--configs, -c` | Config file paths | Required |
| `--combos, -l` | Combo list file | Required |
| `--proxies, -p` | Proxy list file | Optional |
| `--output, -o` | Output directory | `results` |
| `--workers, -w` | Number of workers | `100` |
| `--auto-scrape` | Auto-scrape proxies | `true` |
| `--valid-only` | Save only valid results | `true` |
| `--request-timeout` | Request timeout (ms) | `30000` |
| `--proxy-timeout` | Proxy timeout (ms) | `5000` |

## Variable Substitution

The following variables are automatically replaced in configurations:

| Variable | Description |
|----------|-------------|
| `<USER>` or `<username>` | Username from combo |
| `<PASS>` or `<password>` | Password from combo |
| `<EMAIL>` or `<email>` | Email from combo |

## Live Statistics

The checker displays real-time statistics including:
- Current CPM (Checks Per Minute)
- Valid/Invalid/Error counts
- Progress bar
- Active workers
- Working proxies count
- Elapsed time

## Output

Results are saved in the specified output directory with the following structure:
```
results/
├── valid.txt      # Valid combos
├── invalid.txt    # Invalid combos (if --valid-only=false)
├── errors.txt     # Error combos (if --valid-only=false)
└── stats.json     # Final statistics
```

## Performance Tips

1. **Optimize Workers**: Start with 100 workers and adjust based on your system and target
2. **Use Good Proxies**: Quality proxies improve success rates and reduce errors
3. **Configure Timeouts**: Lower timeouts for faster checking, higher for reliability
4. **Monitor CPM**: Aim for 300-1000+ CPM depending on target and configuration

## License

This project is open source and available under the MIT License.
