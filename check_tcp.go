package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"time"
)

func main() {
	timeout := flag.Float64("t", 1, "timeout waiting for response")
	showAll := flag.Bool("a", false, "show all servers, including OK status")
	quiet := flag.Bool("q", false, "quietly exit when no servers in cmdline")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [<options>...] <servers>...\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()
	if 0 == len(flag.Args()) {
		if *quiet {
			os.Exit(0)
		} else {
			flag.Usage()
		}
	}

	set := make(map[string]struct{}, len(flag.Args()))
	for _, srv := range flag.Args() {
		set[srv] = struct{}{}
	}
	ch := make(chan response, len(set))
	for srv, _ := range set {
		go check(ch, srv)
	}
	tch := make(chan bool)
	go sendTimeout(tch, *timeout)

	allOK := true
	for 0 != len(set) {
		select {
		case got := <-ch:
			delete(set, got.addr)
			if !got.ok {
				fmt.Printf("TCP %s fail: %s\n", got.addr, got.err)
				allOK = false
			}
			if got.ok && *showAll {
				fmt.Printf("TCP %s OK\n", got.addr)
			}
		case <-tch:
			for srv, _ := range set {
				fmt.Printf("TCP %s fail: timeout\n", srv)
			}
			os.Exit(1)
		}
	}
	if !allOK {
		os.Exit(1)
	}
}

type response struct {
	addr string
	err  string
	ok   bool
}

func check(ch chan response, srv string) {
	srvWithPort := srv
	if hasColon, _ := regexp.MatchString(`:`, srv); !hasColon {
		srvWithPort = srv + ":9"
	}
	if ok, _ := regexp.MatchString(`^\d+\.\d+\.\d+\.\d+:\d+$`, srvWithPort); !ok {
		ch <- response{ok: false, addr: srv, err: "bad server, must be <ip> or <ip>:<port>"}
		return
	}
	_, err := net.Dial("tcp", srvWithPort)
	if nil == err {
		//ch <-response{addr: srv, err: "connected", ok: false}
		ch <- response{ok: true, addr: srv}
	} else if err.Error() == "dial tcp "+srvWithPort+": connection refused" {
		ch <- response{ok: true, addr: srv}
	} else {
		ch <- response{addr: srv, err: err.Error(), ok: false}
	}
}

func sendTimeout(tch chan bool, t float64) {
	time.Sleep(time.Duration(t * float64(time.Second)))
	tch <- true
}

func warnf(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Warning: "+f+"\n", args...)
}

func dief(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+f+"\n", args...)
	os.Exit(1)
}
