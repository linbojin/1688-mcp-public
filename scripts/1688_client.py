#!/usr/bin/env python3
"""
1688-MCP command-line client

Usage:
  python3 scripts/1688_client.py status                    # Check login status
  python3 scripts/1688_client.py search <keyword> [-n N]   # Search products
  python3 scripts/1688_client.py puhuo <url>               # List product to Douyin shop
"""

import argparse
import json
import sys
import urllib.request

BASE_URL = "http://localhost:18688"


def cmd_status(args):
    req = urllib.request.Request(
        f"{BASE_URL}/api/v1/login/status",
        headers={"Content-Type": "application/json"},
        method="GET",
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            result = json.loads(resp.read())
    except urllib.error.URLError as e:
        print(f"Request failed: {e}")
        print(f"   Is 1688-mcp running at {BASE_URL}?")
        return 1

    if result.get("success"):
        data = result.get("data", {})
        if data.get("logged_in"):
            username = data.get("username", "")
            print(f"✓ Logged in" + (f" as {username}" if username else ""))
        else:
            print("✗ Not logged in — run 1688-login to authenticate")
    else:
        print(f"Status check failed: {result.get('error', 'unknown error')}")
        return 1
    return 0


def cmd_search(args):
    data = json.dumps({"keyword": args.keyword, "count": args.limit}).encode()
    req = urllib.request.Request(
        f"{BASE_URL}/api/v1/search",
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            result = json.loads(resp.read())
    except urllib.error.URLError as e:
        print(f"Request failed: {e}")
        print(f"   Is 1688-mcp running at {BASE_URL}?")
        return 1

    if result.get("success"):
        products = result.get("data", [])
        print(f"Found {len(products)} products:\n")
        for i, p in enumerate(products, 1):
            print(f"  {i}. {p.get('title', '(no title)')}")
            print(f"     Price: {p.get('price', '?')}  Supplier: {p.get('supplier', '?')}")
            print(f"     URL: {p.get('detail_url', '?')}")
            tags = []
            if p.get("free_shipping"):
                tags.append("包邮")
            if p.get("one_dropship"):
                tags.append("一件代发")
            if p.get("supports_douyin"):
                tags.append("抖音密文面单")
            if tags:
                print(f"     Tags: {', '.join(tags)}")
            print()
    else:
        print(f"Search failed: {result.get('error', 'unknown error')}")
        return 1
    return 0


def cmd_puhuo(args):
    data = json.dumps({"url": args.url}).encode()
    req = urllib.request.Request(
        f"{BASE_URL}/api/v1/puhuo",
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            result = json.loads(resp.read())
    except urllib.error.URLError as e:
        print(f"Request failed: {e}")
        print(f"   Is 1688-mcp running at {BASE_URL}?")
        return 1

    if result.get("success"):
        puhuo_data = result.get("data", {})
        if puhuo_data.get("success"):
            print(f"✓ Puhuo success: {puhuo_data.get('message', '')}")
        else:
            print(f"✗ Puhuo failed: {puhuo_data.get('message', '')}")
            return 1
    else:
        print(f"Puhuo failed: {result.get('error', 'unknown error')}")
        return 1
    return 0


def main():
    parser = argparse.ArgumentParser(description="1688-MCP CLI")
    sub = parser.add_subparsers(dest="command")

    sub.add_parser("status", help="Check login status")

    sp_search = sub.add_parser("search", help="Search products")
    sp_search.add_argument("keyword", help="Search keyword")
    sp_search.add_argument("-n", "--limit", type=int, default=10, help="Max results")

    sp_puhuo = sub.add_parser("puhuo", help="List product to Douyin shop")
    sp_puhuo.add_argument("url", help="Product detail URL (detail.1688.com/offer/...)")

    args = parser.parse_args()
    if not args.command:
        parser.print_help()
        return 1

    handlers = {"status": cmd_status, "search": cmd_search, "puhuo": cmd_puhuo}
    return handlers[args.command](args)


if __name__ == "__main__":
    exit(main())
