#!/usr/bin/python3
# coding: utf-8

import simplejson
import subprocess
"""
    chrome build 628235 works.

    Why host binding fails on newer builds:
    https://github.com/chromium/chromium/commit/d31383577e0517843c8059dec9b87469bf30900f#diff-d717572478f6a97f889b33917c9d3a5f

    Find older builds:
    https://github.com/macchrome/winchrome/releases?after=v77.0.3865.90-r681094-Win64

    Download:
    https://storage.googleapis.com/chromium-browser-snapshots/Linux_x64/628235/chrome-linux.zip
"""


def main():
    target = "http://176.28.50.165/"
    headers = {
        "Host": "testphp.vulnweb.com",
        "User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) "
                      "Chrome/74.0.3945.0 Safari/537.36",
    }
    cmd = ["./rad", "--chrome", "/tmp/chrome-linux-628235/chrome",
           "-o", "json", "--custom-headers", simplejson.dumps(headers), "-t", target]
    rsp = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    output, error = rsp.communicate()

    result = simplejson.loads(output.decode().split("--[Mission Complete]--")[1])
    req_list = result["req_list"]
    for each in req_list:
        print(each)


if __name__ == '__main__':
    main()





