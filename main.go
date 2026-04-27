package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const Version = "v1.0.1"

func handleT(w http.ResponseWriter, r *http.Request) {
	dc, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "err", http.StatusInternalServerError)
		return
	}
	cc, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go tx(dc, cc)
	go tx(cc, dc)
}

func tx(d io.WriteCloser, s io.ReadCloser) {
	defer d.Close()
	defer s.Close()
	io.Copy(d, s)
}

func handleH(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	b := flag.String("b", ":44129", "")
	a := flag.String("a", "", "")
	v := flag.Bool("v", false, "")
	flag.Parse()

	if *v {
		fmt.Println(Version)
		return
	}

	s := &http.Server{
		Addr: *b,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if *a != "" {
				h := r.URL.Hostname()
				if h == "" {
					h = r.Host
					if strings.Contains(h, ":") {
						h = strings.Split(h, ":")[0]
					}
				}
				ok := false
				for _, d := range strings.Split(*a, ",") {
					if strings.HasSuffix(h, strings.TrimSpace(d)) {
						ok = true
						break
					}
				}
				if !ok {
					http.Error(w, "err", http.StatusForbidden)
					return
				}
			}

			if r.Method == http.MethodConnect {
				handleT(w, r)
			} else {
				handleH(w, r)
			}
		}),
	}

	if err := s.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}
