//go:build js && wasm

package wasmsdk

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"syscall/js"
	"time"
)

// NewWebSocketPeerNet creates a peer network that dials Lightning peer TCP
// addresses through a WebSocket byte-stream proxy. The proxy receives the target
// peer address in a "target" query parameter.
func NewWebSocketPeerNet(proxyURL string) (*WebSocketPeerNet, error) {
	if proxyURL == "" {
		return nil, errors.New("empty peer proxy URL")
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return nil, fmt.Errorf("peer proxy URL must use ws or wss")
	}

	return &WebSocketPeerNet{proxyURL: parsed}, nil
}

// WebSocketPeerNet implements tor.Net for browser-hosted peer connections.
type WebSocketPeerNet struct {
	proxyURL *url.URL
}

// Dial connects to a peer through the configured WebSocket proxy.
func (w *WebSocketPeerNet) Dial(network, address string,
	timeout time.Duration) (net.Conn, error) {

	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("unsupported browser peer network: %s",
			network)
	}

	u := *w.proxyURL
	q := u.Query()
	q.Set("target", address)
	u.RawQuery = q.Encode()
	js.Global().Get("console").Call("info", "lnd wasm peer websocket dial",
		map[string]any{
			"target": address,
			"url":    u.String(),
		},
	)

	return newWebSocketConn(u.String(), address, timeout)
}

// LookupHost resolves numeric hosts. Browser DNS resolution is intentionally
// left to the WebSocket proxy.
func (w *WebSocketPeerNet) LookupHost(host string) ([]string, error) {
	if net.ParseIP(host) == nil {
		return nil, fmt.Errorf("browser peer DNS requires proxy: %s", host)
	}

	return []string{host}, nil
}

// LookupSRV is not available in the browser peer network.
func (w *WebSocketPeerNet) LookupSRV(service, proto, name string,
	timeout time.Duration) (string, []*net.SRV, error) {

	return "", nil, errors.New("browser peer SRV lookup requires proxy")
}

// ResolveTCPAddr resolves numeric TCP addresses without host DNS lookup.
func (w *WebSocketPeerNet) ResolveTCPAddr(network,
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

type websocketAddr string

func (w websocketAddr) Network() string {
	return "websocket"
}

func (w websocketAddr) String() string {
	return string(w)
}

type webSocketConn struct {
	ws js.Value

	remote net.Addr

	mu           sync.Mutex
	readDeadline time.Time
	writeClosed  bool
	readBuf      []byte

	readCh  chan []byte
	errCh   chan error
	closeCh chan struct{}

	callbacks []wsCallback
}

type wsCallback struct {
	event string
	fn    js.Func
}

func newWebSocketConn(rawURL, target string,
	timeout time.Duration) (*webSocketConn, error) {

	ws := js.Global().Get("WebSocket").New(rawURL)
	ws.Set("binaryType", "arraybuffer")

	remoteAddr, err := net.ResolveTCPAddr("tcp", target)
	if err != nil {
		remoteAddr = nil
	}
	if remoteAddr == nil {
		remoteAddr = &net.TCPAddr{
			IP: net.IPv4(127, 0, 0, 1),
		}
	}

	conn := &webSocketConn{
		ws:      ws,
		remote:  remoteAddr,
		readCh:  make(chan []byte, 32),
		errCh:   make(chan error, 1),
		closeCh: make(chan struct{}),
	}

	openCh := make(chan error, 1)

	conn.addCallback("open", func(js.Value, []js.Value) any {
		js.Global().Get("console").Call("info",
			"lnd wasm peer websocket open", rawURL,
		)
		select {
		case openCh <- nil:
		default:
		}
		return nil
	})
	conn.addCallback("error", func(js.Value, []js.Value) any {
		js.Global().Get("console").Call("error",
			"lnd wasm peer websocket error", rawURL,
		)
		select {
		case openCh <- errors.New("websocket peer proxy error"):
		default:
		}
		conn.closeWithError(errors.New("websocket peer proxy error"))
		return nil
	})
	conn.addCallback("close", func(js.Value, []js.Value) any {
		js.Global().Get("console").Call("warn",
			"lnd wasm peer websocket close", rawURL,
		)
		conn.closeWithError(io.EOF)
		return nil
	})
	conn.addCallback("message", func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}

		data := args[0].Get("data")
		bytes := uint8ArrayFrom(data)
		select {
		case conn.readCh <- bytes:
		case <-conn.closeCh:
		}

		return nil
	})

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-openCh:
		if err != nil {
			_ = conn.Close()
			return nil, err
		}

		return conn, nil

	case <-timer.C:
		_ = conn.Close()
		return nil, fmt.Errorf("websocket peer proxy dial timed out")
	}
}

func (w *webSocketConn) Read(p []byte) (int, error) {
	for {
		w.mu.Lock()
		if len(w.readBuf) > 0 {
			n := copy(p, w.readBuf)
			w.readBuf = w.readBuf[n:]
			w.mu.Unlock()
			return n, nil
		}

		deadline := w.readDeadline
		w.mu.Unlock()

		var timer <-chan time.Time
		if !deadline.IsZero() {
			wait := time.Until(deadline)
			if wait <= 0 {
				return 0, osErrTimeout{}
			}
			timer = time.After(wait)
		}

		select {
		case bytes := <-w.readCh:
			w.mu.Lock()
			w.readBuf = append(w.readBuf, bytes...)
			w.mu.Unlock()

		case err := <-w.errCh:
			return 0, err

		case <-timer:
			return 0, osErrTimeout{}
		}
	}
}

func (w *webSocketConn) Write(p []byte) (int, error) {
	w.mu.Lock()
	closed := w.writeClosed
	w.mu.Unlock()
	if closed {
		return 0, net.ErrClosed
	}

	data := js.Global().Get("Uint8Array").New(len(p))
	js.CopyBytesToJS(data, p)
	w.ws.Call("send", data)
	return len(p), nil
}

func (w *webSocketConn) Close() error {
	w.mu.Lock()
	if w.writeClosed {
		w.mu.Unlock()
		return nil
	}
	w.writeClosed = true
	w.mu.Unlock()

	for _, cb := range w.callbacks {
		w.ws.Call("removeEventListener", cb.event, cb.fn)
		cb.fn.Release()
	}

	w.ws.Call("close")
	w.closeWithError(net.ErrClosed)

	return nil
}

func (w *webSocketConn) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP: net.IPv4(127, 0, 0, 1),
	}
}

func (w *webSocketConn) RemoteAddr() net.Addr {
	return w.remote
}

func (w *webSocketConn) SetDeadline(t time.Time) error {
	_ = w.SetReadDeadline(t)
	return w.SetWriteDeadline(t)
}

func (w *webSocketConn) SetReadDeadline(t time.Time) error {
	w.mu.Lock()
	w.readDeadline = t
	w.mu.Unlock()
	return nil
}

func (w *webSocketConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (w *webSocketConn) addCallback(event string,
	fn func(js.Value, []js.Value) any) {

	cb := js.FuncOf(fn)
	w.callbacks = append(w.callbacks, wsCallback{
		event: event,
		fn:    cb,
	})
	w.ws.Call("addEventListener", event, cb)
}

func (w *webSocketConn) closeWithError(err error) {
	select {
	case <-w.closeCh:
		return

	default:
		close(w.closeCh)
	}

	select {
	case w.errCh <- err:
	default:
	}
}

func uint8ArrayFrom(value js.Value) []byte {
	if value.InstanceOf(js.Global().Get("ArrayBuffer")) {
		array := js.Global().Get("Uint8Array").New(value)
		bytes := make([]byte, array.Get("byteLength").Int())
		js.CopyBytesToGo(bytes, array)
		return bytes
	}

	buffer := value.Get("buffer")
	offset := value.Get("byteOffset").Int()
	length := value.Get("byteLength").Int()
	array := js.Global().Get("Uint8Array").New(buffer, offset, length)
	bytes := make([]byte, length)
	js.CopyBytesToGo(bytes, array)
	return bytes
}

type osErrTimeout struct{}

func (osErrTimeout) Error() string {
	return "i/o timeout"
}

func (osErrTimeout) Timeout() bool {
	return true
}

func (osErrTimeout) Temporary() bool {
	return true
}
