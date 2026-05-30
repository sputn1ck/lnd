//go:build js && wasm

package wasmsdk

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync/atomic"

	"github.com/jessevdk/go-flags"
	"github.com/lightningnetwork/lnd"
	"github.com/lightningnetwork/lnd/chanbackup"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/autopilotrpc"
	"github.com/lightningnetwork/lnd/lnrpc/chainrpc"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"github.com/lightningnetwork/lnd/lnrpc/peersrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lnrpc/signrpc"
	"github.com/lightningnetwork/lnd/lnrpc/verrpc"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"
	"github.com/lightningnetwork/lnd/lnrpc/watchtowerrpc"
	"github.com/lightningnetwork/lnd/lnrpc/wtclientrpc"
	"github.com/lightningnetwork/lnd/signal"
	"github.com/lightningnetwork/lnd/tor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const defaultBufconnSize = 1024 * 1024

var started atomic.Bool

// StartOptions holds the configuration for a browser-hosted lnd instance.
type StartOptions struct {
	// Args are parsed as regular lnd command line arguments.
	Args []string

	// BufconnSize is the in-memory gRPC listener buffer size. If zero, a
	// sensible default is used.
	BufconnSize int

	// PeerNet overrides the network used for Lightning peer connections.
	// Browser builds use NewAperturePeerNet to bridge raw peer traffic
	// through Aperture's authenticated tcpproxy endpoint.
	PeerNet tor.Net

	// ConfigureImplementation can mutate lnd's implementation config before
	// lnd starts. Browser-hosted subservers use this to register additional
	// gRPC services and aux components on the embedded lnd instance.
	ConfigureImplementation func(*lnd.ImplementationCfg) error

	// UseTLS keeps TLS enabled on the in-memory gRPC listener. The default is
	// plaintext because the browser SDK normally keeps this transport
	// in-process and away from the network.
	UseTLS bool
}

// Instance is a running lnd daemon with RPC access over an in-memory gRPC
// listener.
type Instance struct {
	Config *lnd.Config

	// Conn is an unauthenticated connection that is immediately usable for
	// WalletUnlocker and State RPCs before macaroons exist.
	Conn *grpc.ClientConn

	WalletUnlocker lnrpc.WalletUnlockerClient
	State          lnrpc.StateClient

	listener    *bufconn.Listener
	interceptor signal.Interceptor
	done        chan error
	skipTLS     bool
	stopped     atomic.Bool
	exitErr     atomic.Value
}

// Clients contains the main authenticated RPC clients. The underlying Conn is
// owned by this value and should be closed by the caller.
type Clients struct {
	Conn *grpc.ClientConn

	Lightning        lnrpc.LightningClient
	Autopilot        autopilotrpc.AutopilotClient
	ChainNotifier    chainrpc.ChainNotifierClient
	Invoices         invoicesrpc.InvoicesClient
	Peers            peersrpc.PeersClient
	Router           routerrpc.RouterClient
	Signer           signrpc.SignerClient
	Versioner        verrpc.VersionerClient
	WalletKit        walletrpc.WalletKitClient
	Watchtower       watchtowerrpc.WatchtowerClient
	WatchtowerClient wtclientrpc.WatchtowerClientClient
}

// Close closes the underlying gRPC connection.
func (c *Clients) Close() error {
	if c == nil || c.Conn == nil {
		return nil
	}

	return c.Conn.Close()
}

// Start launches lnd in a goroutine and returns once the RPC listener is ready.
func Start(ctx context.Context, opts StartOptions) (*Instance, error) {
	if !started.CompareAndSwap(false, true) {
		return nil, errors.New("lnd already started")
	}

	bufSize := opts.BufconnSize
	if bufSize == 0 {
		bufSize = defaultBufconnSize
	}

	interceptor, err := signal.Intercept()
	if err != nil {
		started.Store(false)
		return nil, err
	}

	cfg, err := loadConfig(opts.Args, interceptor)
	if err != nil {
		started.Store(false)
		interceptor.RequestShutdown()
		return nil, err
	}

	rpcReady := make(chan struct{})
	listener := bufconn.Listen(bufSize)
	instance := &Instance{
		Config:      cfg,
		listener:    listener,
		interceptor: interceptor,
		done:        make(chan error, 1),
		skipTLS:     !opts.UseTLS,
	}

	listenerCfg := lnd.ListenerCfg{
		SkipTLS:       !opts.UseTLS,
		Net:           opts.PeerNet,
		BackupSwapper: chanbackup.NewMemorySwapper(),
		RPCListeners: []*lnd.ListenerWithSignal{{
			Listener: listener,
			Ready:    rpcReady,
		}},
	}
	implCfg := cfg.ImplementationConfig(interceptor)
	if opts.ConfigureImplementation != nil {
		err := opts.ConfigureImplementation(implCfg)
		if err != nil {
			started.Store(false)
			interceptor.RequestShutdown()
			return nil, err
		}
	}

	go func() {
		defer started.Store(false)

		err := lnd.Main(cfg, listenerCfg, implCfg, interceptor)
		if flagErr, ok := err.(*flags.Error); ok &&
			flagErr.Type == flags.ErrHelp {

			err = nil
		}

		if err != nil {
			instance.exitErr.Store(err)
		}
		instance.stopped.Store(true)

		instance.done <- err
		close(instance.done)
	}()

	select {
	case <-ctx.Done():
		interceptor.RequestShutdown()
		return nil, ctx.Err()

	case err := <-instance.done:
		if err == nil {
			err = errors.New("lnd stopped before RPC became ready")
		}
		return nil, err

	case <-rpcReady:
	}

	conn, err := instance.dial(ctx, true)
	if err != nil {
		interceptor.RequestShutdown()
		return nil, err
	}

	instance.Conn = conn
	instance.WalletUnlocker = lnrpc.NewWalletUnlockerClient(conn)
	instance.State = lnrpc.NewStateClient(conn)

	return instance, nil
}

// AdminClients returns the main RPC clients using admin macaroon credentials.
// Call this after the wallet has been created or unlocked unless lnd is running
// with --no-macaroons.
func (i *Instance) AdminClients(ctx context.Context) (*Clients, error) {
	conn, err := i.dial(ctx, false)
	if err != nil {
		return nil, err
	}

	return NewClients(conn), nil
}

// Dialer returns a gRPC context dialer for the embedded lnd RPC listener.
func (i *Instance) Dialer() func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return i.listener.DialContext(ctx)
	}
}

// NewClients creates typed RPC clients for an existing connection.
func NewClients(conn *grpc.ClientConn) *Clients {
	return &Clients{
		Conn:             conn,
		Lightning:        lnrpc.NewLightningClient(conn),
		Autopilot:        autopilotrpc.NewAutopilotClient(conn),
		ChainNotifier:    chainrpc.NewChainNotifierClient(conn),
		Invoices:         invoicesrpc.NewInvoicesClient(conn),
		Peers:            peersrpc.NewPeersClient(conn),
		Router:           routerrpc.NewRouterClient(conn),
		Signer:           signrpc.NewSignerClient(conn),
		Versioner:        verrpc.NewVersionerClient(conn),
		WalletKit:        walletrpc.NewWalletKitClient(conn),
		Watchtower:       watchtowerrpc.NewWatchtowerClient(conn),
		WatchtowerClient: wtclientrpc.NewWatchtowerClientClient(conn),
	}
}

// Stop requests graceful shutdown.
func (i *Instance) Stop() {
	if i != nil {
		i.interceptor.RequestShutdown()
	}
}

// Wait blocks until lnd exits.
func (i *Instance) Wait() error {
	if i == nil {
		return nil
	}

	return <-i.done
}

// ExitErr returns lnd's exit error if the daemon has already stopped.
func (i *Instance) ExitErr() (error, bool) {
	if i == nil {
		return nil, false
	}

	if !i.stopped.Load() {
		return nil, false
	}

	err, _ := i.exitErr.Load().(error)
	return err, true
}

// Close closes the startup RPC connection and requests shutdown.
func (i *Instance) Close() error {
	if i == nil {
		return nil
	}

	i.Stop()
	if i.Conn == nil {
		return nil
	}

	return i.Conn.Close()
}

func (i *Instance) dial(ctx context.Context,
	skipMacaroons bool) (*grpc.ClientConn, error) {

	var opts []grpc.DialOption
	if i.skipTLS {
		opts = append(
			opts, grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	} else {
		var err error
		opts, err = lnd.AdminAuthOptions(i.Config, skipMacaroons)
		if err != nil {
			return nil, err
		}
	}

	opts = append(opts, grpc.WithContextDialer(i.Dialer()))

	return grpc.DialContext(ctx, "bufconn", opts...)
}

func loadConfig(args []string, interceptor signal.Interceptor) (*lnd.Config,
	error) {

	oldArgs := os.Args
	os.Args = append([]string{"lnd-wasm"}, args...)
	defer func() {
		os.Args = oldArgs
	}()

	cfg, err := lnd.LoadConfig(interceptor)
	if err != nil {
		return nil, fmt.Errorf("load lnd config: %w", err)
	}

	return cfg, nil
}
