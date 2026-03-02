# 1688-mcp

A Go MCP server for 1688 dropship automation — search for dropship-friendly products and list them to your Douyin shop (铺货) with a single command.

**GitHub:** [linbojin/1688-mcp](https://github.com/linbojin/1688-mcp)

## MCP Tools

| Tool | Description |
|------|-------------|
| `check_login` | Verify current login status |
| `refresh_cookies` | Reload cookies from disk after re-login |
| `search_1688` | Search for dropship-friendly products (one-dropship, free shipping, 48H delivery, Douyin secret waybill) |
| `puhuo` | List a product to your Douyin shop |

## Setup

### Step 1: Download Binaries

Download from [GitHub Releases](https://github.com/linbojin/1688-mcp/releases):

| Platform | MCP Server | Login Tool |
|----------|------------|------------|
| macOS (Apple Silicon) | `1688-mcp-darwin-arm64` | `1688-login-darwin-arm64` |
| macOS (Intel) | `1688-mcp-darwin-amd64` | `1688-login-darwin-amd64` |
| Linux | `1688-mcp-linux-amd64` | `1688-login-linux-amd64` |

Grant execute permission:

```shell
chmod +x 1688-mcp-darwin-arm64 1688-login-darwin-arm64
```

### Step 2: Login (First Time Only)

```shell
./1688-login-darwin-arm64
```

A browser window opens. Log in to 1688.com using phone number or Alipay QR code. Cookies are saved automatically once login is detected.

### Step 3: Start the MCP Server

```shell
# Headless mode (recommended)
./1688-mcp-darwin-arm64

# Visible browser for debugging
./1688-mcp-darwin-arm64 -headless=false
```

Server runs at `http://localhost:18688`.

## MCP Client Configuration

Add to your MCP client config (Claude Code, Cursor, etc.):

```json
{
  "mcpServers": {
    "1688-mcp": {
      "type": "http",
      "url": "http://localhost:18688/mcp"
    }
  }
}
```

## REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/api/v1/login/status` | Check login status |
| `POST` | `/api/v1/login/refresh` | Reload cookies from disk |
| `POST` | `/api/v1/search` | Search products `{"keyword": "...", "count": 20}` |
| `POST` | `/api/v1/puhuo` | Puhuo a product `{"url": "https://detail.1688.com/offer/..."}` |

## Python CLI

A Python client is included for quick testing:

```shell
python scripts/1688_client.py status
python scripts/1688_client.py search "车载摆件" -n 5
python scripts/1688_client.py puhuo "https://detail.1688.com/offer/123456789.html"
```

## Build from Source

```shell
make build        # Build for current platform
make build-all    # Cross-compile for all platforms
```

## Re-login

If cookies expire, run the login tool again and then call `refresh_cookies`:

```shell
./1688-login-darwin-arm64
# Then in your MCP client: call refresh_cookies tool
```
