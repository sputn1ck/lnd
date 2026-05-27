package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	ossignal "os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/jessevdk/go-flags"
	"github.com/lightningnetwork/lnd"
	"github.com/lightningnetwork/lnd/chanbackup"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/signal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	bufconnSize = 1024 * 1024
	password    = "lnd-wasm-chain-peer"
)

const (
	bitcoindRPCUser = "user"
	bitcoindRPCPass = "pass"
)

type readyInfo struct {
	Pubkey     string `json:"pubkey"`
	Host       string `json:"host"`
	EsploraURL string `json:"esplora_url"`
}

type command struct {
	ID      string `json:"id"`
	Method  string `json:"method"`
	Pubkey  string `json:"pubkey,omitempty"`
	Host    string `json:"host,omitempty"`
	Address string `json:"address,omitempty"`
	Amount  int64  `json:"amount,omitempty"`
	Invoice string `json:"invoice,omitempty"`
}

type commandResult struct {
	ID              string `json:"id"`
	OK              bool   `json:"ok"`
	Error           string `json:"error,omitempty"`
	Channel         string `json:"channel_point,omitempty"`
	Txid            string `json:"txid,omitempty"`
	PaymentRequest  string `json:"payment_request,omitempty"`
	PaymentHash     string `json:"payment_hash,omitempty"`
	PaymentPreimage string `json:"payment_preimage,omitempty"`
}

type instance struct {
	interceptor signal.Interceptor
	listener    *bufconn.Listener
	done        chan error
}

type bitcoindNode struct {
	name    string
	network string
	rpcHost string
	dataDir string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tempDir, err := os.MkdirTemp("", "lnd-wasm-chain-peer-")
	if err != nil {
		exitErr("create temp dir", err)
	}
	defer os.RemoveAll(tempDir)

	bitcoind, err := startBitcoind(ctx, tempDir)
	if err != nil {
		exitErr("start bitcoind", err)
	}

	electrs, err := startElectrs(ctx, tempDir, bitcoind)
	if err != nil {
		bitcoind.stop()
		exitErr("start electrs", err)
	}
	cleanupDocker := func() {
		electrs.stop()
		bitcoind.stop()
	}
	defer cleanupDocker()

	sigCh := make(chan os.Signal, 1)
	ossignal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer ossignal.Stop(sigCh)
	go func() {
		<-sigCh
		cleanupDocker()
		os.Exit(0)
	}()

	p2pPort, err := freePort()
	if err != nil {
		exitErr("allocate p2p port", err)
	}

	inst, err := startLnd(ctx, defaultArgs(tempDir, p2pPort, bitcoind))
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
	if err := fundWallet(ctx, bitcoind, conn); err != nil {
		exitErr("fund wallet", err)
	}

	info, err := lnrpc.NewLightningClient(conn).GetInfo(
		ctx, &lnrpc.GetInfoRequest{},
	)
	if err != nil {
		exitErr("get info", err)
	}

	host := net.JoinHostPort("127.0.0.1", strconv.Itoa(p2pPort))
	if err := json.NewEncoder(os.Stdout).Encode(readyInfo{
		Pubkey:     info.IdentityPubkey,
		Host:       host,
		EsploraURL: electrs.url,
	}); err != nil {
		exitErr("write peer info", err)
	}

	runCommandLoop(conn, bitcoind)
}

func startBitcoind(ctx context.Context, tempDir string) (*bitcoindNode,
	error) {

	rpcPort, err := freePort()
	if err != nil {
		return nil, err
	}
	name := "lnd-wasm-bitcoind-" + filepath.Base(tempDir)
	network := "lnd-wasm-net-" + filepath.Base(tempDir)
	dataDir := filepath.Join(tempDir, "bitcoind")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	_ = exec.Command("docker", "network", "rm", network).Run()
	if out, err := exec.CommandContext(
		ctx, "docker", "network", "create", network,
	).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("docker network create: %w: %s", err, out)
	}

	removeContainer(name)
	args := []string{
		"run",
		"--rm",
		"--detach",
		"--name", name,
		"--network", network,
		"-p", fmt.Sprintf("127.0.0.1:%d:18443", rpcPort),
		"-v", dataDir + ":/home/bitcoin/.bitcoin",
		"mirror.gcr.io/lightninglabs/bitcoin-core:29",
		"-regtest",
		"-server",
		"-txindex",
		"-fallbackfee=0.0002",
		"-rpcuser=" + bitcoindRPCUser,
		"-rpcpassword=" + bitcoindRPCPass,
		"-rpcbind=0.0.0.0",
		"-rpcallowip=0.0.0.0/0",
		"-dnsseed=0",
		"-upnp=0",
		"-printtoconsole",
	}

	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		_ = exec.Command("docker", "network", "rm", network).Run()
		return nil, fmt.Errorf("docker run bitcoind: %w: %s", err, out)
	}

	node := &bitcoindNode{
		name:    name,
		network: network,
		rpcHost: net.JoinHostPort("127.0.0.1", strconv.Itoa(rpcPort)),
		dataDir: dataDir,
	}
	if err := waitForBitcoind(ctx, node); err != nil {
		node.stop()
		return nil, err
	}
	if err := bitcoindRPC(ctx, node, "createwallet", []any{""}, nil); err != nil {
		return nil, err
	}
	if err := generateBlocks(ctx, node, 150); err != nil {
		return nil, err
	}

	return node, nil
}

type electrsContainer struct {
	name string
	url  string
}

func startElectrs(ctx context.Context, tempDir string,
	bitcoind *bitcoindNode) (*electrsContainer, error) {

	port, err := freePort()
	if err != nil {
		return nil, err
	}

	name := "lnd-wasm-electrs-" + filepath.Base(tempDir)
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	args := []string{
		"run",
		"--rm",
		"--detach",
		"--name", name,
		"--network", bitcoind.network,
		"-p", fmt.Sprintf("127.0.0.1:%d:3002", port),
		"-v", bitcoind.dataDir + ":/home/user/.bitcoin",
		"mirror.gcr.io/mempool/electrs:latest",
		"-vvv",
		"--timestamp",
		"--network=regtest",
		"--cookie=" + bitcoindRPCUser + ":" + bitcoindRPCPass,
		"--daemon-rpc-addr=" + bitcoind.name + ":18443",
		"--http-addr=0.0.0.0:3002",
		"--electrum-rpc-addr=0.0.0.0:60401",
		"--cors=*",
		"--daemon-dir=/home/user/.bitcoin",
		"--db-dir=/home/user/.bitcoin/electrs-db",
	}

	removeContainer(name)
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker run electrs: %w: %s", err, out)
	}

	container := &electrsContainer{
		name: name,
		url:  url,
	}
	if err := waitForEsplora(ctx, url); err != nil {
		container.stop()
		return nil, err
	}

	return container, nil
}

func waitForEsplora(ctx context.Context, url string) error {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		req, err := http.NewRequestWithContext(
			ctx, http.MethodGet, url+"/blocks/tip/height", nil,
		)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
				lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			} else {
				lastErr = err
			}
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("wait for electrs: %w", lastErr)
			}

			return ctx.Err()

		case <-ticker.C:
		}
	}
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func waitForBitcoind(ctx context.Context, node *bitcoindNode) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		lastErr = bitcoindRPC(ctx, node, "getblockcount", nil, nil)
		if lastErr == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for bitcoind: %w", lastErr)

		case <-ticker.C:
		}
	}
}

func generateBlocks(ctx context.Context, node *bitcoindNode, blocks int) error {
	var address string
	if err := bitcoindRPC(
		ctx, node, "getnewaddress", nil, &address,
	); err != nil {
		return err
	}

	return bitcoindRPC(
		ctx, node, "generatetoaddress", []any{blocks, address}, nil,
	)
}

func bitcoindRPC(ctx context.Context, node *bitcoindNode, method string,
	params []any, result any) error {

	if params == nil {
		params = []any{}
	}

	body, err := json.Marshal(rpcRequest{
		JSONRPC: "1.0",
		ID:      "lnd-wasm",
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, "http://"+node.rpcHost+"/",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return err
	}
	req.SetBasicAuth(bitcoindRPCUser, bitcoindRPCPass)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bitcoind RPC %s HTTP %d: %s",
			method, resp.StatusCode, string(respBody))
	}

	var decoded rpcResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return err
	}
	if decoded.Error != nil {
		return fmt.Errorf("bitcoind RPC %s failed (%d): %s",
			method, decoded.Error.Code, decoded.Error.Message)
	}

	if result != nil {
		return json.Unmarshal(decoded.Result, result)
	}

	return nil
}

func (e *electrsContainer) stop() {
	removeContainer(e.name)
}

func (b *bitcoindNode) stop() {
	removeContainer(b.name)
	if b.network != "" {
		_ = exec.Command("docker", "network", "rm", b.network).Run()
	}
}

func removeContainer(name string) {
	cmd := exec.Command("docker", "rm", "-f", name)
	_ = cmd.Run()
}

func startLnd(ctx context.Context, args []string) (*instance, error) {
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
	os.Args = append([]string{"lnd-wasm-chain-peer"}, args...)
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

	_, err = waitForAnyState(
		ctx, stateClient, lnrpc.WalletState_SERVER_ACTIVE,
	)
	return err
}

func fundWallet(ctx context.Context, bitcoind *bitcoindNode,
	conn *grpc.ClientConn) error {

	client := lnrpc.NewLightningClient(conn)
	addrResp, err := client.NewAddress(ctx, &lnrpc.NewAddressRequest{
		Type: lnrpc.AddressType_WITNESS_PUBKEY_HASH,
	})
	if err != nil {
		return err
	}

	if err := bitcoindRPC(
		ctx, bitcoind, "sendtoaddress", []any{addrResp.Address, 1.0}, nil,
	); err != nil {
		return err
	}

	if err := generateBlocks(ctx, bitcoind, 6); err != nil {
		return err
	}

	return waitForBalance(ctx, client, btcutil.SatoshiPerBitcoin/2)
}

func waitForBalance(ctx context.Context, client lnrpc.LightningClient,
	min btcutil.Amount) error {

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		resp, err := client.WalletBalance(ctx, &lnrpc.WalletBalanceRequest{})
		if err != nil {
			return err
		}
		if btcutil.Amount(resp.ConfirmedBalance) >= min {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
		}
	}
}

func runCommandLoop(conn *grpc.ClientConn, bitcoind *bitcoindNode) {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	client := lnrpc.NewLightningClient(conn)
	router := routerrpc.NewRouterClient(conn)
	var acceptCancel context.CancelFunc

	for scanner.Scan() {
		var cmd command
		if err := json.Unmarshal(scanner.Bytes(), &cmd); err != nil {
			writeResult(encoder, commandResult{
				OK:    false,
				Error: err.Error(),
			})
			continue
		}

		switch cmd.Method {
		case "accept_zero_conf":
			if acceptCancel == nil {
				var err error
				acceptCancel, err = acceptZeroConf(client)
				if err != nil {
					writeResult(encoder, commandResult{
						ID:    cmd.ID,
						OK:    false,
						Error: err.Error(),
					})
					continue
				}
			}

			writeResult(encoder, commandResult{
				ID: cmd.ID,
				OK: true,
			})

		case "fund_address":
			txid, err := fundAddress(bitcoind, cmd.Address, cmd.Amount)
			if err != nil {
				writeResult(encoder, commandResult{
					ID:    cmd.ID,
					OK:    false,
					Error: err.Error(),
				})
				continue
			}

			writeResult(encoder, commandResult{
				ID:   cmd.ID,
				OK:   true,
				Txid: txid,
			})

		case "open_zero_conf":
			channel, err := openZeroConf(client, cmd.Pubkey)
			if err != nil {
				writeResult(encoder, commandResult{
					ID:    cmd.ID,
					OK:    false,
					Error: err.Error(),
				})
				continue
			}

			writeResult(encoder, commandResult{
				ID:      cmd.ID,
				OK:      true,
				Channel: channel,
			})

		case "create_invoice":
			invoice, err := createInvoice(client, cmd.Amount, cmd.Pubkey)
			if err != nil {
				writeResult(encoder, commandResult{
					ID:    cmd.ID,
					OK:    false,
					Error: err.Error(),
				})
				continue
			}

			writeResult(encoder, commandResult{
				ID:             cmd.ID,
				OK:             true,
				PaymentRequest: invoice.PaymentRequest,
				PaymentHash:    hex.EncodeToString(invoice.RHash),
			})

		case "pay_invoice":
			payment, err := payInvoice(router, cmd.Invoice)
			if err != nil {
				writeResult(encoder, commandResult{
					ID:    cmd.ID,
					OK:    false,
					Error: err.Error(),
				})
				continue
			}

			writeResult(encoder, commandResult{
				ID:              cmd.ID,
				OK:              true,
				PaymentHash:     payment.PaymentHash,
				PaymentPreimage: payment.PaymentPreimage,
			})

		default:
			writeResult(encoder, commandResult{
				ID:    cmd.ID,
				OK:    false,
				Error: "unknown method",
			})
		}
	}

	if acceptCancel != nil {
		acceptCancel()
	}
}

func fundAddress(bitcoind *bitcoindNode, address string, amount int64) (
	string, error) {

	if address == "" {
		return "", errors.New("fund_address requires address")
	}
	if amount <= 0 {
		amount = 100_000
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	btc := float64(amount) / btcutil.SatoshiPerBitcoin
	var txid string
	if err := bitcoindRPC(
		ctx, bitcoind, "sendtoaddress", []any{address, btc}, &txid,
	); err != nil {
		return "", err
	}

	if err := generateBlocks(ctx, bitcoind, 6); err != nil {
		return "", err
	}

	return txid, nil
}

func acceptZeroConf(client lnrpc.LightningClient) (context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := client.ChannelAcceptor(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				return
			}

			err = stream.Send(&lnrpc.ChannelAcceptResponse{
				Accept:        true,
				PendingChanId: req.PendingChanId,
				ZeroConf:      true,
			})
			if err != nil {
				return
			}
		}
	}()

	return cancel, nil
}

func createInvoice(client lnrpc.LightningClient, amount int64,
	routeHintPeer string) (*lnrpc.AddInvoiceResponse, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	invoice := &lnrpc.Invoice{
		Memo:    "native receive",
		Value:   amount,
		Private: true,
	}

	if routeHintPeer != "" {
		routeHints, err := invoiceRouteHints(ctx, client, routeHintPeer)
		if err != nil {
			return nil, err
		}

		invoice.RouteHints = routeHints
	}

	return client.AddInvoice(ctx, invoice)
}

func invoiceRouteHints(ctx context.Context, client lnrpc.LightningClient,
	peer string) ([]*lnrpc.RouteHint, error) {

	resp, err := client.ListChannels(ctx, &lnrpc.ListChannelsRequest{})
	if err != nil {
		return nil, err
	}

	for _, channel := range resp.Channels {
		if !channel.Active || !strings.EqualFold(channel.RemotePubkey, peer) {
			continue
		}

		return []*lnrpc.RouteHint{{
			HopHints: []*lnrpc.HopHint{{
				NodeId:                    peer,
				ChanId:                    channel.ChanId,
				FeeBaseMsat:               1000,
				FeeProportionalMillionths: 1,
				CltvExpiryDelta:           40,
			}},
		}}, nil
	}

	return nil, fmt.Errorf("active channel for route hint peer %s not found", peer)
}

func payInvoice(client routerrpc.RouterClient,
	invoice string) (*lnrpc.Payment, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	stream, err := client.SendPaymentV2(ctx, &routerrpc.SendPaymentRequest{
		PaymentRequest:    invoice,
		FeeLimitSat:       1000,
		TimeoutSeconds:    60,
		NoInflightUpdates: true,
	})
	if err != nil {
		return nil, err
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil, errors.New("payment stream closed before final state")
		}
		if err != nil {
			return nil, err
		}

		switch resp.Status {
		case lnrpc.Payment_SUCCEEDED:
			return resp, nil

		case lnrpc.Payment_FAILED:
			return nil, fmt.Errorf(
				"payment failed: %v", resp.FailureReason,
			)
		}
	}
}

func openZeroConf(client lnrpc.LightningClient, pubkey string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := waitForPeer(ctx, client, pubkey); err != nil {
		return "", err
	}

	pubBytes, err := hex.DecodeString(pubkey)
	if err != nil {
		return "", err
	}

	stream, err := client.OpenChannel(ctx, &lnrpc.OpenChannelRequest{
		NodePubkey:         pubBytes,
		LocalFundingAmount: 100_000,
		PushSat:            50_000,
		Private:            true,
		MinConfs:           1,
		ZeroConf:           true,
		CommitmentType:     lnrpc.CommitmentType_ANCHORS,
	})
	if err != nil {
		return "", err
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			return "", err
		}

		chanOpen := update.GetChanOpen()
		if chanOpen == nil {
			continue
		}

		return channelPointString(chanOpen.ChannelPoint), nil
	}
}

func waitForPeer(ctx context.Context, client lnrpc.LightningClient,
	pubkey string) error {

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		resp, err := client.ListPeers(ctx, &lnrpc.ListPeersRequest{})
		if err != nil {
			return err
		}

		for _, peer := range resp.Peers {
			if peer.PubKey == pubkey {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
		}
	}
}

func channelPointString(point *lnrpc.ChannelPoint) string {
	if point == nil {
		return ""
	}

	var txid string
	switch fundingTxID := point.FundingTxid.(type) {
	case *lnrpc.ChannelPoint_FundingTxidBytes:
		txid = hex.EncodeToString(fundingTxID.FundingTxidBytes)

	case *lnrpc.ChannelPoint_FundingTxidStr:
		txid = fundingTxID.FundingTxidStr
	}

	return fmt.Sprintf("%s:%d", txid, point.OutputIndex)
}

func writeResult(encoder *json.Encoder, result commandResult) {
	if err := encoder.Encode(result); err != nil {
		exitErr("write command result", err)
	}
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

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

func defaultArgs(tempDir string, p2pPort int,
	bitcoind *bitcoindNode) []string {

	tlsCert := filepath.Join(tempDir, "tls.cert")
	tlsKey := filepath.Join(tempDir, "tls.key")

	return []string{
		"--bitcoin.active",
		"--bitcoin.regtest",
		"--bitcoin.node=bitcoind",
		"--bitcoind.rpchost=" + bitcoind.rpcHost,
		"--bitcoind.rpcuser=" + bitcoindRPCUser,
		"--bitcoind.rpcpass=" + bitcoindRPCPass,
		"--bitcoind.rpcpolling",
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
		"--protocol.option-scid-alias",
		"--protocol.zero-conf",
		"--bitcoin.defaultchanconfs=0",
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
