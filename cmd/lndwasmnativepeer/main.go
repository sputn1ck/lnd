package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/lightningnetwork/lnd"
	"github.com/lightningnetwork/lnd/chanbackup"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/signal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	bufconnSize = 1024 * 1024
	password    = "lnd-wasm-native-peer"
)

type peerInfo struct {
	Pubkey string `json:"pubkey"`
	Host   string `json:"host"`
}

type instance struct {
	cfg         *lnd.Config
	interceptor signal.Interceptor
	listener    *bufconn.Listener
	done        chan error
	exitErr     atomic.Value
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	tempDir, err := os.MkdirTemp("", "lnd-wasm-native-peer-")
	if err != nil {
		exitErr("create temp dir", err)
	}
	defer os.RemoveAll(tempDir)

	p2pPort, err := freePort()
	if err != nil {
		exitErr("allocate p2p port", err)
	}

	inst, err := start(ctx, defaultArgs(tempDir, p2pPort))
	if err != nil {
		exitErr("start lnd", err)
	}
	defer inst.stop()

	conn, err := inst.dial(ctx)
	if err != nil {
		exitErr("dial lnd rpc", err)
	}
	defer conn.Close()

	if err := ensureWallet(ctx, conn); err != nil {
		exitErr("ensure wallet", err)
	}

	info, err := lnrpc.NewLightningClient(conn).GetInfo(
		ctx, &lnrpc.GetInfoRequest{},
	)
	if err != nil {
		exitErr("get info", err)
	}

	host := net.JoinHostPort("127.0.0.1", strconv.Itoa(p2pPort))
	if err := json.NewEncoder(os.Stdout).Encode(peerInfo{
		Pubkey: info.IdentityPubkey,
		Host:   host,
	}); err != nil {
		exitErr("write peer info", err)
	}

	if err := inst.wait(); err != nil {
		exitErr("lnd exited", err)
	}
}

func start(ctx context.Context, args []string) (*instance, error) {
	interceptor, err := signal.Intercept()
	if err != nil {
		return nil, err
	}

	cfg, err := loadConfig(args, interceptor)
	if err != nil {
		interceptor.RequestShutdown()
		return nil, err
	}

	rpcReady := make(chan struct{})
	listener := bufconn.Listen(bufconnSize)
	inst := &instance{
		cfg:         cfg,
		interceptor: interceptor,
		listener:    listener,
		done:        make(chan error, 1),
	}

	listenerCfg := lnd.ListenerCfg{
		SkipTLS:       true,
		BackupSwapper: chanbackup.NewMemorySwapper(),
		RPCListeners: []*lnd.ListenerWithSignal{{
			Listener: listener,
			Ready:    rpcReady,
		}},
	}
	implCfg := cfg.ImplementationConfig(interceptor)

	go func() {
		err := lnd.Main(cfg, listenerCfg, implCfg, interceptor)
		if flagErr, ok := err.(*flags.Error); ok &&
			flagErr.Type == flags.ErrHelp {

			err = nil
		}
		if err != nil {
			inst.exitErr.Store(err)
		}
		inst.done <- err
		close(inst.done)
	}()

	select {
	case <-ctx.Done():
		interceptor.RequestShutdown()
		return nil, ctx.Err()

	case err := <-inst.done:
		if err == nil {
			err = errors.New("lnd stopped before RPC became ready")
		}
		return nil, err

	case <-rpcReady:
		return inst, nil
	}
}

func loadConfig(args []string, interceptor signal.Interceptor) (*lnd.Config,
	error) {

	oldArgs := os.Args
	os.Args = append([]string{"lnd-wasm-native-peer"}, args...)
	defer func() {
		os.Args = oldArgs
	}()

	cfg, err := lnd.LoadConfig(interceptor)
	if err != nil {
		return nil, fmt.Errorf("load lnd config: %w", err)
	}

	return cfg, nil
}

func ensureWallet(ctx context.Context, conn *grpc.ClientConn) error {
	stateClient := lnrpc.NewStateClient(conn)
	unlocker := lnrpc.NewWalletUnlockerClient(conn)

	state, err := waitForAnyState(ctx, stateClient,
		lnrpc.WalletState_NON_EXISTING,
		lnrpc.WalletState_LOCKED,
		lnrpc.WalletState_UNLOCKED,
		lnrpc.WalletState_RPC_ACTIVE,
		lnrpc.WalletState_SERVER_ACTIVE,
	)
	if err != nil {
		return err
	}

	if state == lnrpc.WalletState_NON_EXISTING ||
		state == lnrpc.WalletState_LOCKED {

		seedResp, err := unlocker.GenSeed(ctx, &lnrpc.GenSeedRequest{})
		if err != nil {
			return err
		}

		_, err = unlocker.InitWallet(ctx, &lnrpc.InitWalletRequest{
			WalletPassword:     []byte(password),
			CipherSeedMnemonic: seedResp.CipherSeedMnemonic,
			StatelessInit:      true,
		})
		if err != nil {
			return err
		}
	}

	_, err = waitForAnyState(ctx, stateClient,
		lnrpc.WalletState_RPC_ACTIVE,
		lnrpc.WalletState_SERVER_ACTIVE,
	)
	return err
}

func waitForAnyState(ctx context.Context, client lnrpc.StateClient,
	states ...lnrpc.WalletState) (lnrpc.WalletState, error) {

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		resp, err := client.GetState(ctx, &lnrpc.GetStateRequest{})
		if err != nil {
			return 0, err
		}

		for _, state := range states {
			if resp.State == state {
				return resp.State, nil
			}
		}

		select {
		case <-ctx.Done():
			return resp.State, ctx.Err()

		case <-ticker.C:
		}
	}
}

func (i *instance) dial(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		ctx, "bufconn",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (
			net.Conn, error) {

			return i.listener.DialContext(ctx)
		}),
	)
}

func (i *instance) stop() {
	i.interceptor.RequestShutdown()
}

func (i *instance) wait() error {
	return <-i.done
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

func defaultArgs(tempDir string, p2pPort int) []string {
	tlsCert := filepath.Join(tempDir, "tls.cert")
	tlsKey := filepath.Join(tempDir, "tls.key")

	return []string{
		"--bitcoin.active",
		"--bitcoin.regtest",
		"--bitcoin.node=nochainbackend",
		"--rpclisten=127.0.0.1:0",
		"--restlisten=127.0.0.1:0",
		"--tor.socks=127.0.0.1:9050",
		"--tor.control=127.0.0.1:9051",
		fmt.Sprintf("--listen=127.0.0.1:%d", p2pPort),
		fmt.Sprintf("--externalip=127.0.0.1:%d", p2pPort),
		"--norest",
		"--no-macaroons",
		"--noseedbackup",
		"--nobootstrap",
		"--lnddir=" + tempDir,
		"--logdir=" + tempDir,
		"--tlscertpath=" + tlsCert,
		"--tlskeypath=" + tlsKey,
		"--debuglevel=error",
		"--logging.file.disable",
		"--maxpendingchannels=1",
		fmt.Sprintf("--trickledelay=%d", int64(time.Millisecond)),
	}
}

func exitErr(action string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", action, err)
	os.Exit(1)
}
