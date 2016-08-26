# Marill
Marill -- Automated site testing utility

--------------------------------------------------------

[![Build Status](https://travis-ci.org/Liamraystanley/marill.svg?branch=master)](https://travis-ci.org/Liamraystanley/marill)
[![GitHub Issues](https://img.shields.io/github/issues/Liamraystanley/marill.svg)](https://github.com/Liamraystanley/marill/issues)
[![Project Status](https://img.shields.io/badge/status-pre--alpha-red.svg)](https://github.com/Liamraystanley/marill/commits/master)
[![Codebeat Badge](https://codebeat.co/badges/4653f785-83ec-4b21-bf0c-b519b20c89d6)](https://codebeat.co/projects/github-com-liamraystanley-marill)
[![Go Report Card](https://goreportcard.com/badge/github.com/Liamraystanley/marill)](https://goreportcard.com/report/github.com/Liamraystanley/marill)

#### Project Status:
At this stage, things are still in alpha/likely going to change quite a bit. This includes code, exported functions/tools, cli args, etc. This is what I intend to have completed for the beta release:

- [x] crawling of pages recursively
- [x] scan **Apache**/**Lightspeed** based webservers for domains to fetch
- [x] scan **cPanel** based webservers for domains to fetch
- [ ] scan **Nginx** based webservers for domains to fetch
- [ ] cli arg manager
- [ ] scan any webserver based on manual cli input
- [ ] return results in a human readible format
- [ ] return results in a bot/script readible format
- [ ] provide potential fixes
- [ ] scan server for possible issues (reference error_log files, webserver error logs, etc)


##### License:

    LICENSE: The MIT License (MIT)
    Copyright (c) 2016 Liam Stanley <me@liamstanley.io>

    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:
    
    The above copyright notice and this permission notice shall be included in
    all copies or substantial portions of the Software.
    
    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
    SOFTWARE.
