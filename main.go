package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ypdn/cookiejar"
	"golang.org/x/net/publicsuffix"
)

var (
	method       = flag.String("m", "GET", "http method")
	body         = flag.String("f", "", "request body file")
	verbose      = flag.Bool("v", false, "write status and headers to stderr")
	timeout      = flag.Duration("t", 0, "timeout (0 means no timeout)")
	maxRedirects = flag.Uint("r", 10, "number of redirects to follow")
	exitRedirect = flag.Bool("e", false, "exit with error if the redirection limit is exceeded")
	jarPath      = flag.String("j", filepath.Join(must(os.UserHomeDir()), ".hreq-cookies"), "directory to use as cookie jar (pass empty string to use an in-memory jar)")
)

func main() {
	header := make(http.Header)
	flag.Func("h", "http header", func(s string) error {
		k, v, f := strings.Cut(s, ":")
		if !f {
			return errors.New("bad header format, expecting 'key:value'")
		}
		header.Set(k, v)
		return nil
	})

	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		usage()
	}

	u := must(url.Parse(args[0]))
	r := &http.Request{
		Method:     *method,
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Host:       u.Host,
	}
	if *body != "" {
		r.Body = must(os.Open(*body))
	}
	jar := must(cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
		Directory:        *jarPath,
		ErrorLog:         log.New(os.Stderr, "cookiejar: ", 0),
	}))
	c := &http.Client{
		Timeout:       *timeout,
		CheckRedirect: checkRedirect,
		Jar:           jar,
	}
	resp := must(c.Do(r))
	defer resp.Body.Close()

	if *verbose {
		must(fmt.Fprintln(os.Stderr, resp.StatusCode, http.StatusText(resp.StatusCode), "\n"))
		check(resp.Header.Write(os.Stderr))
	}
	must(io.Copy(os.Stdout, resp.Body))
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > int(*maxRedirects) {
		if *exitRedirect {
			return fmt.Errorf("stopped after %v redirects", *maxRedirects)
		}
		return http.ErrUseLastResponse
	}
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %v [flags] url\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func must[T any](t T, err error) T {
	check(err)
	return t
}
