package http

import (
	"context"
	"net"
	"net/http"
	"net/http/httptrace"

	"github.com/sirupsen/logrus"
)

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"100.64.0.0/10",  // RFC6598
		"172.16.0.0/12",  // RFC1918
		"192.0.0.0/24",   // RFC6890
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func blocksContain(blocks []*net.IPNet, ip net.IP) bool {
	for _, block := range blocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	return blocksContain(privateIPBlocks, ip)
}

type noLocalTransport struct {
	inner         http.RoundTripper
	errlog        logrus.FieldLogger
	allowedBlocks []*net.IPNet
}

func (no noLocalTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(req.Context())

	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			addr := info.Conn.RemoteAddr().String()
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				cancel()
				no.errlog.WithError(err).Error("Cancelled request due to error in address parsing")
				return
			}
			ip := net.ParseIP(host)
			if ip == nil {
				cancel()
				no.errlog.WithError(err).Error("Cancelled request due to error in ip parsing")
				return
			}

			if blocksContain(no.allowedBlocks, ip) {
				return
			}

			if isPrivateIP(ip) {
				cancel()
				no.errlog.Error("Cancelled attempted request to ip in private range")
				return
			}
		},
	})

	req = req.WithContext(ctx)
	return no.inner.RoundTrip(req)
}

func SafeRoundtripper(trans http.RoundTripper, log logrus.FieldLogger, allowedBlocks ...*net.IPNet) http.RoundTripper {
	if trans == nil {
		trans = http.DefaultTransport
	}

	ret := &noLocalTransport{
		inner:         trans,
		errlog:        log.WithField("transport", "local_blocker"),
		allowedBlocks: allowedBlocks,
	}

	return ret
}

func SafeHTTPClient(client *http.Client, log logrus.FieldLogger, allowedBlocks ...*net.IPNet) *http.Client {
	client.Transport = SafeRoundtripper(client.Transport, log, allowedBlocks...)

	return client
}
