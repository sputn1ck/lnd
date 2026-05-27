//go:build js && wasm

package wasmsdk

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	llwasmnet "github.com/lightninglabs/ll-wasm/go/wasmnet"
)

// NewAperturePeerNet creates a peer network that dials Lightning peer TCP
// addresses through Aperture's authenticated tcpproxy service.
func NewAperturePeerNet(proxyURL,
	clientPrivateKeyHex string) (*AperturePeerNet, error) {

	if proxyURL == "" {
		return nil, errors.New("empty aperture peer proxy URL")
	}
	if clientPrivateKeyHex == "" {
		return nil, errors.New("empty aperture client private key")
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return nil, fmt.Errorf("aperture peer proxy URL must use ws or wss")
	}

	net := &AperturePeerNet{
		targetNodePubKeys: make(map[string]string),
	}
	net.dialer = llwasmnet.NewApertureProxyDialer(
		llwasmnet.ApertureProxyConfig{
			URL:                 parsed.String(),
			Profile:             "lnd",
			ClientPrivateKeyHex: clientPrivateKeyHex,
			TargetNodePubKey:    net.targetNodePubKey,
		},
	)

	return net, nil
}

// AperturePeerNet implements tor.Net for browser-hosted Lightning peer
// connections tunneled through Aperture.
type AperturePeerNet struct {
	dialer func(net.Addr) (net.Conn, error)

	mu                sync.RWMutex
	targetNodePubKeys map[string]string
}

// RegisterTargetNodePubKey records the pubkey expected for a peer address.
// lnd's tor.Net interface only receives network and address at Dial time, while
// Aperture's lnd relay profile also needs the target node pubkey.
func (a *AperturePeerNet) RegisterTargetNodePubKey(address,
	pubkey string) {

	a.mu.Lock()
	defer a.mu.Unlock()

	pubkey = strings.ToLower(strings.TrimSpace(pubkey))
	address = strings.TrimSpace(address)
	a.targetNodePubKeys[address] = pubkey

	if host, port, err := net.SplitHostPort(address); err == nil {
		a.targetNodePubKeys[net.JoinHostPort(host, port)] = pubkey
	}
}

// Dial connects to a peer through Aperture's authenticated TCP proxy.
func (a *AperturePeerNet) Dial(network, address string,
	timeout time.Duration) (net.Conn, error) {

	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("unsupported browser peer network: %s",
			network)
	}

	if _, _, err := net.SplitHostPort(address); err != nil {
		return nil, err
	}
	targetAddr := peerProxyAddr(address)

	done := make(chan struct {
		conn net.Conn
		err  error
	}, 1)
	go func() {
		conn, err := a.dialer(targetAddr)
		done <- struct {
			conn net.Conn
			err  error
		}{conn: conn, err: err}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-done:
		return result.conn, result.err

	case <-timer.C:
		return nil, fmt.Errorf("aperture peer proxy dial timed out")
	}
}

// LookupHost resolves numeric hosts. Browser DNS resolution is intentionally
// left to Aperture.
func (a *AperturePeerNet) LookupHost(host string) ([]string, error) {
	if net.ParseIP(host) == nil {
		return nil, fmt.Errorf("browser peer DNS requires proxy: %s", host)
	}

	return []string{host}, nil
}

// LookupSRV is not available in the browser peer network.
func (a *AperturePeerNet) LookupSRV(service, proto, name string,
	timeout time.Duration) (string, []*net.SRV, error) {

	return "", nil, errors.New("browser peer SRV lookup requires proxy")
}

// ResolveTCPAddr resolves numeric TCP addresses without host DNS lookup.
func (a *AperturePeerNet) ResolveTCPAddr(network,
	address string) (*net.TCPAddr, error) {

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	if net.ParseIP(host) == nil {
		return nil, fmt.Errorf("browser peer DNS requires proxy: %s", host)
	}

	return net.ResolveTCPAddr(network, net.JoinHostPort(host, port))
}

func (a *AperturePeerNet) targetNodePubKey(addr net.Addr) string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.targetNodePubKeys[addr.String()]
}

type peerProxyAddr string

func (p peerProxyAddr) Network() string {
	return "tcp"
}

func (p peerProxyAddr) String() string {
	return string(p)
}
