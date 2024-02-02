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
	mFlag = flag.String("m", "get", "http method")
	fFlag = flag.String("f", "", "request body file")
	vFlag = flag.Bool("v", false, "write status and headers to stderr")
	tFlag = flag.Duration("t", 0, "timeout (0 means no timeout)")
	rFlag = flag.Uint("r", 10, "number of redirects to follow")
	eFlag = flag.Bool("e", false, "exit with error if the redirection limit is exceeded")
	jFlag = flag.String("j", filepath.Join(must(os.UserHomeDir()), ".hreq-cookies"), "directory to use as cookie jar (pass empty string to use an in-memory jar)")
)

func main() {
	r := must(http.NewRequest("", "", nil))

	flag.Func("h", "http header", func(s string) error {
		k, v, f := strings.Cut(s, ":")
		if !f {
			return errors.New("bad header format, expecting 'key:value'")
		}
		r.Header.Set(k, v)
		return nil
	})
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		usage()
	}

	r.URL = must(url.Parse(args[0]))
	r.Method = strings.ToUpper(*mFlag)
	if *fFlag != "" {
		r.Body = must(os.Open(*fFlag))
	}
	jar := must(cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
		Directory:        *jFlag,
		ErrorLog:         log.New(os.Stderr, "", 0),
	}))
	c := &http.Client{
		Timeout:       *tFlag,
		CheckRedirect: checkRedirect,
		Jar:           jar,
	}
	resp := must(c.Do(r))
	defer resp.Body.Close()

	if *vFlag {
		must(fmt.Fprintln(os.Stderr, resp.StatusCode, http.StatusText(resp.StatusCode), "\n"))
		check(resp.Header.Write(os.Stderr))
	}
	must(io.Copy(os.Stdout, resp.Body))
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > int(*rFlag) {
		if !*eFlag {
			return http.ErrUseLastResponse
		}
		return fmt.Errorf("stopped after %v redirects", *rFlag)
	}
	return nil
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "usage: %v [flags] url\n", os.Args[0])
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
