package xff

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

// list of private subnets
var privateMasks, _ = toMasks([]string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"fc00::/7",
})

// converts a list of subnets' string to a list of net.IPNet.
func toMasks(ips []string) (masks []net.IPNet, err error) {
	for _, cidr := range ips {
		var network *net.IPNet
		_, network, err = net.ParseCIDR(cidr)
		if err != nil {
			return
		}
		masks = append(masks, *network)
	}
	return
}

// checks if a net.IP is in a list of net.IPNet
func ipInMasks(ip net.IP, masks []net.IPNet) bool {
	for _, mask := range masks {
		if mask.Contains(ip) {
			return true
		}
	}
	return false
}

// IsPublicIP returns true if the given IP can be routed on the Internet.
func IsPublicIP(ip net.IP) bool {
	if !ip.IsGlobalUnicast() {
		return false
	}
	return !ipInMasks(ip, privateMasks)
}

// Parse parses the value of the X-Forwarded-For Header and returns the IP address.
func Parse(ipList string) string {
	for _, ip := range strings.Split(ipList, ",") {
		ip = strings.TrimSpace(ip)
		if IP := net.ParseIP(ip); IP != nil && IsPublicIP(IP) {
			return ip
		}
	}
	return ""
}

// GetRemoteAddr parses the given request, resolves the X-Forwarded-For header
// and returns the resolved remote address.
func GetRemoteAddr(r *http.Request) string {
	return GetRemoteAddrIfAllowed(r, func(sip string) bool { return true })
}

// GetRemoteAddrIfAllowed parses the given request, resolves the X-Forwarded-For header
// and returns the resolved remote address if allowed.
func GetRemoteAddrIfAllowed(r *http.Request, allowed func(sip string) bool) string {
	if xffh := r.Header.Get("X-Forwarded-For"); xffh != "" {
		if sip, sport, err := net.SplitHostPort(r.RemoteAddr); err == nil && sip != "" {
			if allowed(sip) {
				if xip := Parse(xffh); xip != "" {
					return net.JoinHostPort(xip, sport)
				}
			}
		}
	}
	return r.RemoteAddr
}

// Options is a configuration container to setup the XFF middleware.
type Options struct {
	// AllowedSubnets is a list of Subnets from which we will accept the
	// X-Forwarded-For header.
	// If this list is empty we will accept every Subnets (default).
	AllowedSubnets []string
	// Debugging flag adds additional output to debug server side XFF issues.
	Debug bool
}

// XFF http handler
type XFF struct {
	// Debug logger
	Log *log.Logger
	// Set to true if all IPs or Subnets are allowed.
	allowAll bool
	// List of IP subnets that are allowed.
	allowedMasks []net.IPNet
}

// New creates a new XFF handler with the provided options.
func New(options Options) (*XFF, error) {
	allowedMasks, err := toMasks(options.AllowedSubnets)
	if err != nil {
		return nil, err
	}
	xff := &XFF{
		allowAll:     len(options.AllowedSubnets) == 0,
		allowedMasks: allowedMasks,
	}
	if options.Debug {
		xff.Log = log.New(os.Stdout, "[xff] ", log.LstdFlags)
	}
	return xff, nil
}

// Default creates a new XFF handler with default options.
func Default() (*XFF, error) {
	return New(Options{})
}

// Handler updates RemoteAdd from X-Fowarded-For Headers.
func (xff *XFF) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = GetRemoteAddrIfAllowed(r, xff.allowed)
		h.ServeHTTP(w, r)
	})
}

// Negroni compatible interface
func (xff *XFF) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	r.RemoteAddr = GetRemoteAddrIfAllowed(r, xff.allowed)
	next(w, r)
}

// HandlerFunc provides Martini compatible handler
func (xff *XFF) HandlerFunc(w http.ResponseWriter, r *http.Request) {
	r.RemoteAddr = GetRemoteAddrIfAllowed(r, xff.allowed)
}

// checks that the IP is allowed.
func (xff *XFF) allowed(sip string) bool {
	if xff.allowAll {
		return true
	} else if ip := net.ParseIP(sip); ip != nil && ipInMasks(ip, xff.allowedMasks) {
		return true
	}
	return false
}

// convenience method. checks if debugging is turned on before printing
func (xff *XFF) logf(format string, a ...interface{}) {
	if xff.Log != nil {
		xff.Log.Printf(format, a...)
	}
}
