![Marill -- Automated Site Testing Utility](https://i.imgur.com/HYZ3biA.png)
<p align="center">Marill -- Automated site testing utility.</p>

[![Build Status](https://travis-ci.org/lrstanley/marill.svg?branch=master)](https://travis-ci.org/lrstanley/marill)
[![GitHub Issues](https://img.shields.io/github/issues/lrstanley/marill.svg)](https://github.com/lrstanley/marill/issues)
[![Project Status](https://img.shields.io/badge/status-alpha-red.svg)](https://github.com/lrstanley/marill/commits/master)
[![Codebeat Badge](https://codebeat.co/badges/05580d8e-4a2f-4664-b25b-a6c053233982)](https://codebeat.co/projects/github-com-lrstanley-marill)
[![Go Report Card](https://goreportcard.com/badge/github.com/lrstanley/marill)](https://goreportcard.com/report/github.com/lrstanley/marill)

Marill is an automated site testing utility, which is meant to make
administrators lives easier by taking much of the leg-work out of testing.
It's intended to be lightweight, flexible, and easy to use, while still being
very powerful.

## Table of Contents
- [Goal](#goal)
- [Features](#features)
- [Limitations](#limitations)
- [How does it work?](#how-does-it-work)
  - [Examples](#examples)
- [Usage](#usage)
- [Getting started](#getting-started)
  - [cPanel/Apache based servers](#cpanelapache-based-servers)
  - [Alternatives (Nginx, Caddy, etc)](#alternatives-nginx-caddy-etc)
  - [Troubleshooting](#things-to-notetroubleshooting)
- [Frequently Asked Questions](#faq)
  - [Will it cause high load?](#faq)
  - [How long does Marill take to crawl sites (e.g. 1,000 sites on a server)?](#faq)
  - [Is it better to run Marill from inside of the server, or from a remote location?](#faq)
  - [Can I give Marill a custom IP address for which to crawl a site (before it goes live and DNS is updated)?](#faq)
  - [Can I give Marill a custom port for which to crawl a site?](#faq)
  - [Can Marill crawl sub-domains and sub-folders?](#faq)
- [Building](#building)
- [Project Status](#project-status)
- [Contributing](#contributing)

## Goal

Often times during server administration, migrations, and large server changes,
things can and will go wrong. Servers are complex systems with many working
parts, and with that comes a lot of breakage.

Creating an automated site testing utility, like Marill, allows:
   * Less human interaction to test sites.
   * Integration and flexibility to be built into other systems.
   * Clients to be at ease; you know they won't test all of their sites.
   * Administrators or developers to hate you less.
   * You to be witty and say "time is money!".

## Features

_Disclaimer: Marill is still in early development, and this list is subject
to change drastically. (code, libraries, tools, cli-args, etc)_

   * Cross platform: Marill can compile across many platforms. linux, netbsd,
   openbsd, freebsd, and more. (Windows too, possibly!)
   * Configurable output. Only output what you need.
   * Many cli flags to configure input, output, what is tested, what isn't, etc.
   * Ability to test cPanel based servers, Apache, Nginx (coming soon!), and
   others! (any can be scanned with `--domains`)
   * Flexible testing system. You can even write your own tests! Load them
   from a URL in JSON format, or from a directory! (see
   [marill/tests](https://github.com/lrstanley/marill/tree/master/tests))

## Limitations

There are a few limitations with Marill, due to how the utility was developed.
Marill was meant to be lightweight, and portable. This means it cannot work
exactly like a normal browser. Below are a few examples:

   * Marill currently isn't able to take a screenshot of the site. However,
   there many other external resources for this. (usecase: pixel-by-pixel
   comparison -- easily tell if CSS is broken)
   * Marill can't execute Javascript. If your site is heavy on Javascript,
   this tool may not be best suited. (there are some sites which rely heavily
   on Javascript. (however not frequently do sites Javascript break during
   a migration or move, unless a resource fails to load, which Marill should
   catch)
   * Marill cannot and will not load certain resources. E.g. videos, iframes,
   embeds, ftp links, etc. This would make crawling the site very complex and
   convoluted. (however, embed plugins within things like wordpress could
   possibly be caught, if a test were to be written to search for bad tags)
   * Marill cannot search through webserver, PHP, or other misc. logs to
   determine what the issue may be. This will likely never change, because
   adding this functionality would make the utility fragile and clunky. If
   there is an error, you should be able to find out what is causing it.

## How does it work?

The general idea is that you place Marill on the server you would like to
test. Marill by default will then figure out the list of domains that server
is hosting. Marill will then begin to act much like a browser, crawling each
site (and all resources like images/css/javascript/etc if `--assets` is used).
It will then pass each resource it fetches through the list of builtin, or
external tests. Each domain is given a starting score of 10, and each test
has a pre-defined weight. If the test matches, that score is applied to the
main score. If the score falls below the minimum configured score, it is
considered failed.

### Examples

Here are a few examples of tests that are useful:

   * Visual PHP errors on the page. For example:
   `Warning: Invalid argument supplied for function() in /path/to/some/file.php`
   * Invalid status codes. For example, `Forbidden`, `Internal Service Error`,
   `Payment Required`
   * Blank pages generated by PHP (common if PHP has `display_errors` disabled)
   * MySQL or PostgreSQL related errors.
   * cPanel "Sorry!" related pages (common if the incorrect IP is configured
   for example).

Example running from my workstation (though, this would be best suited running
from the server itself):
[![asciicast](https://asciinema.org/a/bhnskk1s3vwdwgl2w52deioel.png)](https://asciinema.org/a/bhnskk1s3vwdwgl2w52deioel)

## Usage
This is very likely to change quite a bit until we're out of beta. Please use
wisely.

```
$ marill --help
NAME:
   marill - Automated website testing utility

USAGE:
   marill [global options] command [command options] [arguments...]

VERSION:
   git revision XXXXXX

AUTHOR(S):
   Liam Stanley <user@domain.com>

COMMANDS:
     scan           [DEFAULT] Start scan for all domains on server
     urls, domains  Print the list of urls as if they were going to be scanned
     tests          Print the list of tests that are loaded and would be used
     help, h        Shows a list of commands or help for one command

GLOBAL OPTIONS:
   -d, --debug              Print debugging information to stdout
   -q, --quiet              Do not print regular stdout messages
   --no-color               Do not print with color
   --no-banner              Do not print the colorful banner
   --show-warnings          Show a warning if one or more test failed, even if it didn't drop below min-score
   --exit-on-fail           Send exit code 1 if any domains fail tests
   --log FILE               Log information to FILE
   --debug-log FILE         Log debugging information to FILE
   --result-file FILE       Dump result template into FILE (will overwrite!)
   --no-updates             Don't check to see if there are updates
   --threads n              Use n threads to fetch data (0 defaults to server cores/2) (default: 0)
   --delay DURATION         Delay DURATION before each resource is crawled (e.g. 5s, 1m, 100ms) (default: 0s)
   --http-timeout DURATION  DURATION before an http request is timed out (e.g. 5s, 10s, 1m) (default: 10s)
   --domains DOMAIN:IP ...  Manually specify list of domains to scan in form: DOMAIN:IP ..., or DOMAIN:IP:PORT
   --min-score value        Minimum score for domain (default: 8)
   -a, --assets             Crawl assets (css/js/images) for each page
   --ignore-success         Only print results if they are considered failed
   --allow-insecure         Don't check to see if an SSL certificate is valid
   --tmpl value             Golang text/template string template for use with formatting scan output
   --json PATH              Optional PATH to output json results to
   --json-pretty            Used with [--json], pretty-prints the output json
   --ignore-http            Ignore http-based URLs during domain search
   --ignore-https           Ignore https-based URLs during domain search
   --ignore-remote          Ignore all resources that resolve to a remote IP (use with --assets)
   --ignore-domains GLOB    Ignore URLS during domain search that match GLOB, pipe separated list
   --match-domains GLOB     Allow URLS during domain search that match GLOB, pipe separated list
   --ignore-test GLOB       Ignore tests that match GLOB, pipe separated list
   --match-test GLOB        Allow tests that match GLOB, pipe separated list
   --tests-url URL          Import tests from a specified URL
   --tests-path PATH        Import tests from a specified file-system PATH
   --ignore-std-tests       Ignores all built-in tests (useful with --tests-url)
   --pass-text GLOB         Give sites a +10 score if body matches GLOB
   --fail-text GLOB         Give sites a -10 score if body matches GLOB
   --help, -h               show help
   --version, -v            print the version

COPYRIGHT:
   (c) 2016 Liam Stanley
```

## Getting Started

Getting started with Marill should be fairly easy. Since Marill is a single
binary, there are no dependencies that are needed for the utility to run.

Head to [this page](https://release.liam.sh/marill/?sort=time&order=desc) and
download the top item in the list. For example, using the latest version:

```bash
$ wget -q -O- https://release.liam.sh/marill/latest.tar.gz | tar -zx -C /root/tmp/
```

You should now see a file named **marill** in `/root/tmp/`. Feel free to look
over the current flags and arguments:

```bash
$ /root/tmp/marill --help
```

The main arguments that may be useful are:
   * `-a` or `--assets`: This will fetch all of the assets for the page
   (css/javascript/images, etc)
   * `-d` or `--debug`: This will enable debugging. It doesn't provide a whole
   lot more information, but can help if something isn't working.
   * `--delay`: Utilize this if the load caused by the crawling is too high.
   E.g. `--delay 10s`.
   * `--threads`: This is the amount of parallel scans that will run at a
   single time. By default it will be 1/2 the amount of cores on the server.
   * `--ignore-domains` and `--match-domains`: utilize these to skip or only
   scan certain domains during the crawl. E.g.
   `--ignore-domains "*domain.com|someotherdomain.com"`

So, for example, to start off with:

```bash
$ /root/tmp/marill -a
```

### cPanel/Apache based servers

Marill has out of the box support for cPanel based servers (though things like
`/var/cpanel/users/<user>` and `/var/cpanel/userdata/<domain>`).

For Apache, Marill will find the current running httpd instance, and run
`<binary> -S`, which pulls information about all virtual host entries. Note
that this isn't supported on all Apache versions (see
[here](https://httpd.apache.org/docs/2.4/vhosts/) for more information).

### Alternatives (Nginx, Caddy, etc)

If your web server does not match the above description, you can utilize the
manual domain list flag of Marill. The current syntax for this is as follows:

```bash
$ marill --domains "<items>"
```

Replace **\<items\>** with one of the following list of inputs:
   * `DOMAIN:IP:PORT`
   * `DOMAIN:IP`
   * `DOMAIN:PORT`
   * `DOMAIN`

**DOMAIN** can be any of one of the following examples:
   * `domain.com`
   * `www.domain.com`
   * `random.subdomain.domain.com`
   * `http://some-example.com/`
   * `https://some-example.com/some-login.php`

So, to put it all together, you can do something like:

```bash
$ marill a --domains "somedomain.com:443 domain.com:1234 example.com:123.456.7.89:80 https://domain.com/"
```

### Things to note/Troubleshooting
   * If there are any problems or bugs, **PLEASE LET ME KNOW!** You can submit
   bugs if you have a Github account [here](https://github.com/lrstanley/marill/issues/new)
   or [here if you do not](https://links.ml/iWQz)

## FAQ
   1. **Will it cause high load?**
      * The general target at which this was written for are servers under
      maintenance, or being ran on a new server that is being migrated to.
      That being said, Marill does run scans in parallel. It will run scans
      in parallel in the amount of cores divided by 2. (8 core server, 4
      concurrent crawls, 2 core server, 1 crawl at a time). If you see Marill
      still causing too much load, you can utilize `--delay` and `--threads`.

   2. **How long does Marill take to crawl sites (e.g. 1,000 sites on a server)?**
      * Given a cPanel server, is must be noted that along with the input
      (default http) version of a domain, the https version of the site will
      be scanned as well if cPanel has a certificate for it. Furthermore, it
      will also attempt to crawl www.domain.com, not just domain.com. As for
      other webservers, it all depends on the input. **Please note** that
      using `--assets` (`-a`), that Marill _will take longer_. This is because
      this fetches all resources for each site being crawled. If you would
      like Marill to crawl faster, don't use `-a`.
      * Generally speaking, crawling without `-a` is fairly fast.

   3. **Is it better to run Marill from inside of the server, or from a remote location?**
      * Running remotely ensures there are no ip or firewall related issues,
      however in the same sense if you are crawling quite a few sites, many
      servers may assume due to the high connection count, that your
      connections are malicious.
      * If ran from inside of the server, Marill can scan and determine what
      the server is hosting (by checking Apache, cPanel, etc).

   4. **Can I give Marill a custom IP address for which to crawl a site (beforeit goes live and DNS is updated)?**
      * **Yes!** For example, rather than `--domains "domain.com domain2.com"`,
      you can do something like:

      ```bash
      $ marill --domains "domain.com:1.2.3.4 domain2.com:2.3.4.5"
      ```

      * Also note that you can run scans on alternate ports:

      ```bash
      $ marill --domains "domain.com:1.2.3.4:8080 domain.com:9000"
      ```

   5. **Can I give Marill a custom port for which to crawl a site?**
      * **Yes!** see **FAQ #4**

   6. **Can Marill crawl sub-domains and sub-folders?**
      * **Yes!** You can pass any url into `--domains` as necessary. For example:

      ```bash
      $ marill --domains "https://domain.com/sub/folder/some-page"
      ```

      * or with a custom ip as well:

      ```bash
      $ marill --domains "https://domain.com/sub/folder/some-page:1.2.3.4"
      ```

## Building
Marill supports building on 1.6+ (or even possibly older), however it is
recommended to build on the latest go release. Note that you will not be able
to use the Makefile to compile Marill if you are trying to build on go 1.4
or lower. You will need to manually compile it, due to limitations with ldflag
support.

```bash
$ git clone https://github.com/lrstanley/marill.git
$ cd marill
$ make help
$ make build
```

## Project Status

   * [See here](https://github.com/lrstanley/marill/projects/1) for what is
   being worked on/in my todo list for the first beta release.
   * [See here](https://github.com/lrstanley/marill/projects/2) for what is
   being worked on/in my todo list for the first major release.
   * Head over to [release.liam.sh/marill](https://release.liam.sh/marill/?sort=time&order=desc)
   to get some testing bundled binaries, if you're daring and willing to help
   test my latest pushes.
   * Head over to [Github Releases](https://github.com/lrstanley/marill/releases)
   to get more true-tested builds and versions, with change information.

## Contributing

Please review the [CONTRIBUTING](https://github.com/lrstanley/marill/blob/master/CONTRIBUTING.md)
doc for submitting issues/a guide on submitting pull requests and helping out.

## License

```
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
```
