# Marill
Marill -- Automated site testing utility

--------------------------------------------------------

[![Build Status](https://travis-ci.org/Liamraystanley/marill.svg?branch=master)](https://travis-ci.org/Liamraystanley/marill)
[![GitHub Issues](https://img.shields.io/github/issues/Liamraystanley/marill.svg)](https://github.com/Liamraystanley/marill/issues)
[![Project Status](https://img.shields.io/badge/status-alpha-red.svg)](https://github.com/Liamraystanley/marill/commits/master)
[![Codebeat Badge](https://codebeat.co/badges/4653f785-83ec-4b21-bf0c-b519b20c89d6)](https://codebeat.co/projects/github-com-liamraystanley-marill)
[![Go Report Card](https://goreportcard.com/badge/github.com/Liamraystanley/marill)](https://goreportcard.com/report/github.com/Liamraystanley/marill)

#### Project Status:
At this stage, things are still in alpha/likely going to change quite a bit. This includes code, exported functions/tools, cli args, etc. This is what I intend to have completed for the beta release:

- [x] crawling of pages recursively
- [x] scan **Apache**/**Lightspeed** based webservers for domains to fetch
- [x] scan **cPanel** based webservers for domains to fetch
- [ ] scan **Nginx** based webservers for domains to fetch
- [X] cli arg manager [**ACTIVELY BEING MODIFIED**]
- [ ] scan any webserver based on manual cli input [**IN PROGRESS**]
- [ ] return results in a human readible format
- [ ] return results in a bot/script readible format

Things I would like to have completed for the first major release:

- [ ] provide potential fixes
- [ ] scan server for possible issues (reference error_log files, webserver error logs, etc)


#### Building:
Marill supports building on 1.3+ (or even possibly older), however it is recommended to build on the latest go release. Note that you will not be able to use the Makefile to compile Marill if you are trying to build on go 1.4 or lower. You will need to manually compile it, due to limitations with ldflag support.

```
$ git clone https://github.com/Liamraystanley/marill.git
$ cd marill
$ make
```

To run unit tests, then compile, simply run:

```
$ make test all
```

#### Usage:
This is very likely to change quite a bit until we're out of beta. Please use wisely.

```
$ marill --help
NAME:
   marill - Automated website testing utility

USAGE:
   marill [global options] command [command options] [arguments...]
   
VERSION:
   git revision XXXXXX
   
AUTHOR(S):
   Liam Stanley <me@liamstanley.io> 
   
COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug, -d           Print debugging information to stdout
   --quiet, -q           Do not print regular stdout messages
   --no-color            Do not print with color
   --log-file logfile    Log debugging information to logfile
   --cores n             Use n cores to fetch data (0 being server cores/2) (default: 0)
   --print-urls          Print the list of urls as if they were going to be scanned
   --ignore-http         Ignore http-based URLs during domain search
   --ignore-https        Ignore https-based URLs during domain search
   --domain-ignore GLOB  Ignore URLS during domain search that match GLOB
   --domain-match GLOB   Allow URLS during domain search that match GLOB
   --recursive           Check all assets (css/js/images) for each page, recursively
   --help, -h            show help
   --version, -v         print the version
```

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
