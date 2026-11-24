#!/usr/bin/python3
# coding: utf-8

import simplejson
import subprocess
"""
    Example: crawling with a Cookie.

    From the command line:
    ./rad --chrome /home/test/chrome-linux/chrome -o json --ignore-url-keywords quit,exit,logout --custom-headers "{\"Cookie\": \"rad=Cool\"}"

    Use --ignore-url-keywords to exclude keywords such as logout links so the
    crawl never hits them.
"""


def main():
    target = "http://testphp.vulnweb.com/"
    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) "
                      "Chrome/74.0.3945.0 Safari/537.36",
        "Cookie": "rad=Cool"
    }
    cmd = ["./rad", "--chrome", "/home/test/chrome-linux/chrome",
           "-o", "json", "--ignore-url-keywords", "quit,exit,logout", "--custom-headers", simplejson.dumps(headers),
           "-t", target]

    rsp = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    output, error = rsp.communicate()

    result = simplejson.loads(output.decode().split("--[Mission Complete]--")[1])
    req_list = result["req_list"]
    for each in req_list:
        print(each)


if __name__ == '__main__':
    main()





