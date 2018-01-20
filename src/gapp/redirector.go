package goengine

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
)

type Prox struct {
	// target url of reverse proxy
	target *url.URL
	// instance of Go ReverseProxy thatwill do the job for us
	proxy *ReverseProxy
}

func invalidUA(userAgent string) bool {
	ua := ""
	if len(ua) != 0 && ua != userAgent {
		return true
	}
	return false
}

func invalidIP(remoteIP string) bool {
	subnet := ""
	if len(subnet) != 0 {
		_, cidr, err := net.ParseCIDR(subnet)
		if err != nil {
			return false
		}
		if !cidr.Contains(net.IP([]byte(remoteIP))) {
			return true
		}
	}
	return false
}

//TODO: Implement header access on the object
func invalidHeader(remoteHeader http.Header) bool {
	header, value := "", ""
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
	return &Prox{target: url, proxy: NewSingleHostReverseProxy(url)}
}

func (p *Prox) handle(w http.ResponseWriter, r *http.Request) {
	encoding := ""
	// call to magic method from ReverseProxy object
	fmt.Println("Request Received")
	if invalidUA(r.UserAgent()) || invalidIP(r.RemoteAddr) || invalidHeader(r.Header) {
		http.Redirect(w, r, "https://google.com", 301)
	} else {
		if len(encoding) != 0 {
			r.Header.Set("Accept-Encoding", encoding)
		}
		p.proxy.ServeHTTP(w, r)
	}
}

func init() {
	// proxy
	// proxy
	proxy := &Prox{}
	proxy = New("http://192.81.213.231/")

	// server
	http.HandleFunc("/", proxy.handle)
}
