//go:build js && wasm

package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"syscall/js"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/wasmsdk"
)

var instance *wasmsdk.Instance
var acceptorCancel context.CancelFunc

func main() {
	js.Global().Set("lndWasmDefaultArgs", js.FuncOf(defaultArgs))
	js.Global().Set("lndWasmStart", js.FuncOf(start))
	js.Global().Set("lndWasmEnsureWallet", js.FuncOf(ensureWallet))
	js.Global().Set("lndWasmGetInfo", js.FuncOf(getInfo))
	js.Global().Set("lndWasmBalances", js.FuncOf(balances))
	js.Global().Set("lndWasmNewAddress", js.FuncOf(newAddress))
	js.Global().Set("lndWasmSendCoins", js.FuncOf(sendCoins))
	js.Global().Set("lndWasmConnectPeer", js.FuncOf(connectPeer))
	js.Global().Set("lndWasmListPeers", js.FuncOf(listPeers))
	js.Global().Set("lndWasmOpenChannel", js.FuncOf(openChannel))
	js.Global().Set(
		"lndWasmAcceptZeroConfChannels", js.FuncOf(acceptZeroConfChannels),
	)
	js.Global().Set("lndWasmListChannels", js.FuncOf(listChannels))
	js.Global().Set("lndWasmAddInvoice", js.FuncOf(addInvoice))
	js.Global().Set("lndWasmPayInvoice", js.FuncOf(payInvoice))
	js.Global().Set(
		"lndWasmWaitInvoiceSettled", js.FuncOf(waitInvoiceSettled),
	)
	js.Global().Set("lndWasmStop", js.FuncOf(stop))

	select {}
}

func defaultArgs(js.Value, []js.Value) any {
	args := js.Global().Get("Array").New()
	for i, arg := range defaultStartArgs() {
		args.SetIndex(i, arg)
	}

	return args
}

func adminClients(ctx context.Context) (*wasmsdk.Clients, error) {
	if instance == nil {
		return nil, fmt.Errorf("lnd is not started")
	}

	clients, err := instance.AdminClients(ctx)
	if err != nil {
		return nil, withDaemonStatus(err)
	}

	return clients, nil
}

func getInfo(js.Value, []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := adminClients(ctx)
		if err != nil {
			reject.Invoke(err.Error())
			return
		}
		defer clients.Close()

		info, err := clients.Lightning.GetInfo(ctx, &lnrpc.GetInfoRequest{})
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		result := js.Global().Get("Object").New()
		result.Set("identityPubkey", info.IdentityPubkey)
		result.Set("alias", info.Alias)
		result.Set("blockHeight", info.BlockHeight)
		result.Set("numPeers", info.NumPeers)
		result.Set("numActiveChannels", info.NumActiveChannels)
		result.Set("syncedToChain", info.SyncedToChain)
		result.Set("chains", len(info.Chains))
		resolve.Invoke(result)
	})
}

func start(_ js.Value, args []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if instance != nil {
			reject.Invoke("lnd already started")
			return
		}

		startArgs := defaultStartArgs()
		if len(args) > 0 && !args[0].IsUndefined() && !args[0].IsNull() {
			startArgs = jsArrayToStrings(args[0])
		}

		ctx, cancel := context.WithTimeout(
			context.Background(), 3*time.Minute,
		)
		defer cancel()

		opts := wasmsdk.StartOptions{
			Args: startArgs,
		}
		peerProxyURL := js.Global().Get("lndWasmPeerProxyURL")
		if peerProxyURL.Type() == js.TypeString && peerProxyURL.String() != "" {
			peerNet, err := wasmsdk.NewWebSocketPeerNet(
				peerProxyURL.String(),
			)
			if err != nil {
				reject.Invoke(err.Error())
				return
			}
			opts.PeerNet = peerNet
		}

		inst, err := wasmsdk.Start(ctx, opts)
		if err != nil {
			reject.Invoke(err.Error())
			return
		}
		instance = inst

		stateResp, err := inst.State.GetState(ctx, &lnrpc.GetStateRequest{})
		if err != nil {
			reject.Invoke(err.Error())
			return
		}

		result := js.Global().Get("Object").New()
		result.Set("rpcReady", true)
		result.Set("walletState", stateResp.State.String())
		resolve.Invoke(result)
	})
}

func connectPeer(_ js.Value, args []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if instance == nil {
			reject.Invoke("lnd is not started")
			return
		}
		if len(args) < 2 {
			reject.Invoke("connect peer requires pubkey and host")
			return
		}

		ctx, cancel := context.WithTimeout(
			context.Background(), 3*time.Minute,
		)
		defer cancel()

		reportPhase("connect-peer")
		js.Global().Get("console").Call("info", "lnd wasm connect peer",
			map[string]any{
				"pubkey": args[0].String(),
				"host":   args[1].String(),
			},
		)
		reportPhase("wait-server-active")
		_, err := waitForAnyState(ctx, lnrpc.WalletState_SERVER_ACTIVE)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		clients, err := instance.AdminClients(ctx)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}
		defer clients.Close()

		resp, err := clients.Lightning.ConnectPeer(
			ctx, &lnrpc.ConnectPeerRequest{
				Addr: &lnrpc.LightningAddress{
					Pubkey: args[0].String(),
					Host:   args[1].String(),
				},
				Timeout: 15,
			},
		)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		peerCount, err := waitForPeer(ctx, clients, args[0].String())
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		result := js.Global().Get("Object").New()
		result.Set("status", resp.Status)
		result.Set("numPeers", peerCount)
		resolve.Invoke(result)
	})
}

func listPeers(js.Value, []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := adminClients(ctx)
		if err != nil {
			reject.Invoke(err.Error())
			return
		}
		defer clients.Close()

		resp, err := clients.Lightning.ListPeers(ctx, &lnrpc.ListPeersRequest{})
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		peers := js.Global().Get("Array").New(len(resp.Peers))
		for i, peer := range resp.Peers {
			item := js.Global().Get("Object").New()
			item.Set("pubkey", peer.PubKey)
			item.Set("address", peer.Address)
			item.Set("inbound", peer.Inbound)
			item.Set("bytesSent", fmt.Sprintf("%d", peer.BytesSent))
			item.Set("bytesRecv", fmt.Sprintf("%d", peer.BytesRecv))
			peers.SetIndex(i, item)
		}

		result := js.Global().Get("Object").New()
		result.Set("peers", peers)
		resolve.Invoke(result)
	})
}

func openChannel(_ js.Value, args []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if len(args) < 2 {
			reject.Invoke("open channel requires pubkey and amount")
			return
		}

		pubBytes, err := hex.DecodeString(args[0].String())
		if err != nil {
			reject.Invoke(err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := adminClients(ctx)
		if err != nil {
			reject.Invoke(err.Error())
			return
		}
		defer clients.Close()

		req := &lnrpc.OpenChannelRequest{
			NodePubkey:         pubBytes,
			LocalFundingAmount: int64(args[1].Int()),
			SatPerVbyte:        1,
			Private:            true,
			MinConfs:           0,
			SpendUnconfirmed:   true,
			CommitmentType:     lnrpc.CommitmentType_ANCHORS,
		}
		if len(args) > 2 && args[2].Type() == js.TypeObject {
			opts := args[2]
			if !opts.Get("pushSat").IsUndefined() {
				req.PushSat = int64(opts.Get("pushSat").Int())
			}
			if !opts.Get("private").IsUndefined() {
				req.Private = opts.Get("private").Bool()
			}
			if !opts.Get("zeroConf").IsUndefined() {
				req.ZeroConf = opts.Get("zeroConf").Bool()
			}
			if !opts.Get("minConfs").IsUndefined() {
				req.MinConfs = int32(opts.Get("minConfs").Int())
			}
			if !opts.Get("satPerVbyte").IsUndefined() {
				req.SatPerVbyte = uint64(opts.Get("satPerVbyte").Int())
			}
		}

		stream, err := clients.Lightning.OpenChannel(ctx, req)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		for {
			update, err := stream.Recv()
			if err != nil {
				reject.Invoke(withDaemonStatus(err).Error())
				return
			}

			chanOpen := update.GetChanOpen()
			if chanOpen == nil {
				continue
			}

			result := js.Global().Get("Object").New()
			result.Set("channelPoint", channelPointString(chanOpen.ChannelPoint))
			resolve.Invoke(result)
			return
		}
	})
}

func acceptZeroConfChannels(js.Value, []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if instance == nil {
			reject.Invoke("lnd is not started")
			return
		}
		if acceptorCancel != nil {
			resolve.Invoke(true)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		clients, err := instance.AdminClients(ctx)
		if err != nil {
			cancel()
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		stream, err := clients.Lightning.ChannelAcceptor(ctx)
		if err != nil {
			_ = clients.Close()
			cancel()
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		acceptorCancel = func() {
			cancel()
			_ = clients.Close()
		}

		go func() {
			defer func() {
				acceptorCancel = nil
				_ = clients.Close()
			}()

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

		resolve.Invoke(true)
	})
}

func balances(js.Value, []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := adminClients(ctx)
		if err != nil {
			reject.Invoke(err.Error())
			return
		}
		defer clients.Close()

		wallet, err := clients.Lightning.WalletBalance(
			ctx, &lnrpc.WalletBalanceRequest{},
		)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		channel, err := clients.Lightning.ChannelBalance(
			ctx, &lnrpc.ChannelBalanceRequest{},
		)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		result := js.Global().Get("Object").New()
		result.Set("totalBalance", wallet.TotalBalance)
		result.Set("confirmedBalance", wallet.ConfirmedBalance)
		result.Set("unconfirmedBalance", wallet.UnconfirmedBalance)
		result.Set("lockedBalance", wallet.LockedBalance)
		result.Set("channelLocalBalance", amountSat(channel.LocalBalance))
		result.Set("channelRemoteBalance", amountSat(channel.RemoteBalance))
		resolve.Invoke(result)
	})
}

func newAddress(_ js.Value, args []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := adminClients(ctx)
		if err != nil {
			reject.Invoke(err.Error())
			return
		}
		defer clients.Close()

		addrType := lnrpc.AddressType_WITNESS_PUBKEY_HASH
		if len(args) > 0 {
			addrType = addressType(args[0].String())
		}

		resp, err := clients.Lightning.NewAddress(
			ctx, &lnrpc.NewAddressRequest{Type: addrType},
		)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		result := js.Global().Get("Object").New()
		result.Set("address", resp.Address)
		resolve.Invoke(result)
	})
}

func sendCoins(_ js.Value, args []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if len(args) < 2 {
			reject.Invoke("send coins requires address and amount")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := adminClients(ctx)
		if err != nil {
			reject.Invoke(err.Error())
			return
		}
		defer clients.Close()

		req := &lnrpc.SendCoinsRequest{
			Addr:                  args[0].String(),
			Amount:                int64(args[1].Int()),
			SatPerVbyte:           1,
			MinConfs:              1,
			SpendUnconfirmed:      false,
			CoinSelectionStrategy: lnrpc.CoinSelectionStrategy_STRATEGY_USE_GLOBAL_CONFIG,
		}
		if len(args) > 2 && args[2].Type() == js.TypeObject {
			opts := args[2]
			if !opts.Get("satPerVbyte").IsUndefined() {
				req.SatPerVbyte = uint64(opts.Get("satPerVbyte").Int())
			}
			if !opts.Get("sendAll").IsUndefined() {
				req.SendAll = opts.Get("sendAll").Bool()
			}
			if !opts.Get("spendUnconfirmed").IsUndefined() {
				req.SpendUnconfirmed = opts.Get("spendUnconfirmed").Bool()
			}
			if !opts.Get("minConfs").IsUndefined() {
				req.MinConfs = int32(opts.Get("minConfs").Int())
			}
		}

		resp, err := clients.Lightning.SendCoins(ctx, req)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		result := js.Global().Get("Object").New()
		result.Set("txid", resp.Txid)
		resolve.Invoke(result)
	})
}

func listChannels(js.Value, []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if instance == nil {
			reject.Invoke("lnd is not started")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := instance.AdminClients(ctx)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}
		defer clients.Close()

		resp, err := clients.Lightning.ListChannels(
			ctx, &lnrpc.ListChannelsRequest{},
		)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		channels := js.Global().Get("Array").New(len(resp.Channels))
		for i, channel := range resp.Channels {
			item := js.Global().Get("Object").New()
			item.Set("channelPoint", channel.ChannelPoint)
			item.Set("active", channel.Active)
			item.Set("private", channel.Private)
			item.Set("remotePubkey", channel.RemotePubkey)
			item.Set("localBalance", channel.LocalBalance)
			item.Set("remoteBalance", channel.RemoteBalance)
			item.Set("zeroConf", channel.ZeroConf)
			item.Set("chanId", fmt.Sprintf("%d", channel.ChanId))
			channels.SetIndex(i, item)
		}

		result := js.Global().Get("Object").New()
		result.Set("channels", channels)
		resolve.Invoke(result)
	})
}

func addInvoice(_ js.Value, args []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if instance == nil {
			reject.Invoke("lnd is not started")
			return
		}
		if len(args) < 1 {
			reject.Invoke("add invoice requires amount")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := instance.AdminClients(ctx)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}
		defer clients.Close()

		resp, err := clients.Lightning.AddInvoice(ctx, &lnrpc.Invoice{
			Memo:    "browser receive",
			Value:   int64(args[0].Int()),
			Private: true,
		})
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		result := js.Global().Get("Object").New()
		result.Set("paymentRequest", resp.PaymentRequest)
		result.Set("paymentHash", hex.EncodeToString(resp.RHash))
		resolve.Invoke(result)
	})
}

func payInvoice(_ js.Value, args []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if instance == nil {
			reject.Invoke("lnd is not started")
			return
		}
		if len(args) < 1 {
			reject.Invoke("pay invoice requires payment request")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := instance.AdminClients(ctx)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}
		defer clients.Close()

		req := &lnrpc.SendRequest{
			PaymentRequest: args[0].String(),
			FeeLimit: &lnrpc.FeeLimit{
				Limit: &lnrpc.FeeLimit_Fixed{
					Fixed: 1000,
				},
			},
		}
		if len(args) > 1 && args[1].String() != "" {
			chanID, err := strconv.ParseUint(args[1].String(), 10, 64)
			if err != nil {
				reject.Invoke(err.Error())
				return
			}

			resp, err := payInvoiceToRoute(
				ctx, clients.Lightning, args[0].String(), chanID,
			)
			if err != nil {
				reject.Invoke(withDaemonStatus(err).Error())
				return
			}
			if resp.PaymentError != "" {
				reject.Invoke(resp.PaymentError)
				return
			}

			result := js.Global().Get("Object").New()
			result.Set("paymentHash", hex.EncodeToString(resp.PaymentHash))
			result.Set(
				"paymentPreimage",
				hex.EncodeToString(resp.PaymentPreimage),
			)
			resolve.Invoke(result)
			return
		}

		resp, err := clients.Lightning.SendPaymentSync(ctx, req)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}
		if resp.PaymentError != "" {
			reject.Invoke(resp.PaymentError)
			return
		}

		result := js.Global().Get("Object").New()
		result.Set("paymentHash", hex.EncodeToString(resp.PaymentHash))
		result.Set("paymentPreimage", hex.EncodeToString(resp.PaymentPreimage))
		resolve.Invoke(result)
	})
}

func payInvoiceToRoute(ctx context.Context, client lnrpc.LightningClient,
	invoice string, chanID uint64) (*lnrpc.SendResponse, error) {

	payReq, err := client.DecodePayReq(ctx, &lnrpc.PayReqString{
		PayReq: invoice,
	})
	if err != nil {
		return nil, err
	}

	paymentHash, err := hex.DecodeString(payReq.PaymentHash)
	if err != nil {
		return nil, err
	}

	info, err := client.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	channels, err := client.ListChannels(ctx, &lnrpc.ListChannelsRequest{})
	if err != nil {
		return nil, err
	}

	var channel *lnrpc.Channel
	for _, candidate := range channels.Channels {
		if candidate.ChanId == chanID {
			channel = candidate
			break
		}
	}
	if channel == nil {
		return nil, fmt.Errorf("channel id %d not found", chanID)
	}

	amtMsat := payReq.NumMsat
	if amtMsat == 0 {
		amtMsat = payReq.NumSatoshis * 1000
	}
	amtSat := (amtMsat + 999) / 1000
	expiry := uint32(int64(info.BlockHeight) + payReq.CltvExpiry)

	hop := &lnrpc.Hop{
		ChanId:           chanID,
		ChanCapacity:     channel.Capacity,
		AmtToForward:     amtSat,
		AmtToForwardMsat: amtMsat,
		Expiry:           expiry,
		PubKey:           payReq.Destination,
	}
	if len(payReq.PaymentAddr) > 0 {
		hop.MppRecord = &lnrpc.MPPRecord{
			PaymentAddr:  payReq.PaymentAddr,
			TotalAmtMsat: amtMsat,
		}
	}

	return client.SendToRouteSync(ctx, &lnrpc.SendToRouteRequest{
		PaymentHash: paymentHash,
		Route: &lnrpc.Route{
			TotalTimeLock:      expiry,
			TotalAmt:           amtSat,
			TotalAmtMsat:       amtMsat,
			FirstHopAmountMsat: amtMsat,
			Hops:               []*lnrpc.Hop{hop},
		},
	})
}

func waitInvoiceSettled(_ js.Value, args []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if instance == nil {
			reject.Invoke("lnd is not started")
			return
		}
		if len(args) < 1 {
			reject.Invoke("wait invoice requires payment hash")
			return
		}

		paymentHash, err := hex.DecodeString(args[0].String())
		if err != nil {
			reject.Invoke(err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		clients, err := instance.AdminClients(ctx)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}
		defer clients.Close()

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			resp, err := clients.Lightning.LookupInvoice(
				ctx, &lnrpc.PaymentHash{RHash: paymentHash},
			)
			if err != nil {
				reject.Invoke(withDaemonStatus(err).Error())
				return
			}
			if resp.State == lnrpc.Invoice_SETTLED {
				result := js.Global().Get("Object").New()
				result.Set("state", resp.State.String())
				result.Set("amtPaidSat", resp.AmtPaidSat)
				resolve.Invoke(result)
				return
			}

			select {
			case <-ctx.Done():
				reject.Invoke(withDaemonStatus(ctx.Err()).Error())
				return

			case <-ticker.C:
			}
		}
	})
}

func ensureWallet(_ js.Value, args []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if instance == nil {
			reject.Invoke("lnd is not started")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		password := []byte("lnd-wasm-demo")
		if len(args) > 0 && args[0].String() != "" {
			password = []byte(args[0].String())
		}

		reportPhase("wait-wallet-state")
		state, err := waitForAnyState(ctx,
			lnrpc.WalletState_NON_EXISTING,
			lnrpc.WalletState_LOCKED,
			lnrpc.WalletState_UNLOCKED,
			lnrpc.WalletState_RPC_ACTIVE,
			lnrpc.WalletState_SERVER_ACTIVE,
		)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		if state == lnrpc.WalletState_NON_EXISTING {
			reportPhase("gen-seed")
			seedResp, err := instance.WalletUnlocker.GenSeed(
				ctx, &lnrpc.GenSeedRequest{},
			)
			if err != nil {
				reject.Invoke(withDaemonStatus(err).Error())
				return
			}

			reportPhase("init-wallet")
			_, err = instance.WalletUnlocker.InitWallet(
				ctx, &lnrpc.InitWalletRequest{
					WalletPassword: password,
					CipherSeedMnemonic: seedResp.
						CipherSeedMnemonic,
					StatelessInit: true,
				},
			)
			if err != nil {
				reject.Invoke(withDaemonStatus(err).Error())
				return
			}
		} else if state == lnrpc.WalletState_LOCKED {
			reportPhase("unlock-wallet")
			_, err = instance.WalletUnlocker.UnlockWallet(
				ctx, &lnrpc.UnlockWalletRequest{
					WalletPassword: password,
				},
			)
			if err != nil {
				reject.Invoke(withDaemonStatus(err).Error())
				return
			}
		}

		reportPhase("wait-rpc-active")
		state, err = waitForAnyState(ctx,
			lnrpc.WalletState_RPC_ACTIVE,
			lnrpc.WalletState_SERVER_ACTIVE,
		)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		reportPhase("get-info")
		clients, err := instance.AdminClients(ctx)
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}
		defer clients.Close()

		info, err := clients.Lightning.GetInfo(ctx, &lnrpc.GetInfoRequest{})
		if err != nil {
			reject.Invoke(withDaemonStatus(err).Error())
			return
		}

		result := js.Global().Get("Object").New()
		result.Set("walletState", state.String())
		result.Set("identityPubkey", info.IdentityPubkey)
		result.Set("alias", info.Alias)
		result.Set("blockHeight", info.BlockHeight)
		result.Set("numPeers", info.NumPeers)
		resolve.Invoke(result)
	})
}

func reportPhase(phase string) {
	js.Global().Set("__lndWasmPhase", phase)
	js.Global().Get("console").Call("log", "lnd wasm phase: "+phase)
}

func amountSat(amount *lnrpc.Amount) uint64 {
	if amount == nil {
		return 0
	}

	return amount.Sat
}

func addressType(name string) lnrpc.AddressType {
	switch name {
	case "nested":
		return lnrpc.AddressType_NESTED_PUBKEY_HASH
	case "taproot":
		return lnrpc.AddressType_TAPROOT_PUBKEY
	default:
		return lnrpc.AddressType_WITNESS_PUBKEY_HASH
	}
}

func channelPointString(point *lnrpc.ChannelPoint) string {
	if point == nil {
		return ""
	}

	txid := point.GetFundingTxidStr()
	if txid == "" {
		txid = hex.EncodeToString(point.GetFundingTxidBytes())
	}

	return fmt.Sprintf("%s:%d", txid, point.OutputIndex)
}

func withDaemonStatus(err error) error {
	if exitErr, ok := instance.ExitErr(); ok {
		if exitErr != nil {
			return fmt.Errorf("%w; lnd exited: %v", err, exitErr)
		}

		return fmt.Errorf("%w; lnd exited cleanly", err)
	}

	return fmt.Errorf("%w; lnd still running", err)
}

func waitForPeer(ctx context.Context, clients *wasmsdk.Clients,
	pubkey string) (int, error) {

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		resp, err := clients.Lightning.ListPeers(
			ctx, &lnrpc.ListPeersRequest{},
		)
		if err != nil {
			return 0, err
		}

		for _, peer := range resp.Peers {
			if peer.PubKey == pubkey {
				return len(resp.Peers), nil
			}
		}

		select {
		case <-ctx.Done():
			return len(resp.Peers), ctx.Err()

		case <-ticker.C:
		}
	}
}

func stop(js.Value, []js.Value) any {
	return promise(func(resolve, reject js.Value) {
		if instance == nil {
			resolve.Invoke(true)
			return
		}

		if acceptorCancel != nil {
			acceptorCancel()
			acceptorCancel = nil
		}

		instance.Stop()
		err := instance.Close()
		if err != nil {
			reject.Invoke(err.Error())
			return
		}

		instance = nil
		resolve.Invoke(true)
	})
}

func promise(run func(resolve, reject js.Value)) any {
	handler := js.FuncOf(func(_ js.Value, args []js.Value) any {
		resolve := args[0]
		reject := args[1]

		go run(resolve, reject)
		return nil
	})

	return js.Global().Get("Promise").New(handler)
}

func waitForAnyState(ctx context.Context, states ...lnrpc.WalletState) (
	lnrpc.WalletState, error) {

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		resp, err := instance.State.GetState(ctx, &lnrpc.GetStateRequest{})
		if err != nil {
			return 0, err
		}

		for _, state := range states {
			if resp.State == state {
				return resp.State, nil
			}
		}

		if resp.State == lnrpc.WalletState_WAITING_TO_START {
			select {
			case <-ctx.Done():
				return resp.State, ctx.Err()

			case <-ticker.C:
			}
			continue
		}

		if len(states) == 0 {
			return resp.State, nil
		}

		select {
		case <-ctx.Done():
			return resp.State, ctx.Err()

		case <-ticker.C:
		}
	}
}

func jsArrayToStrings(value js.Value) []string {
	length := value.Get("length").Int()
	args := make([]string, 0, length)
	for i := 0; i < length; i++ {
		args = append(args, value.Index(i).String())
	}

	return args
}

func defaultStartArgs() []string {
	network := "regtest"
	networkValue := js.Global().Get("lndWasmBitcoinNetwork")
	if networkValue.Type() == js.TypeString && networkValue.String() != "" {
		network = networkValue.String()
	}

	chainArgs := []string{
		"--bitcoin.active",
		"--bitcoin." + network,
	}

	esploraURL := js.Global().Get("lndWasmEsploraURL")
	if esploraURL.Type() == js.TypeString && esploraURL.String() != "" {
		chainArgs = append(
			chainArgs,
			"--bitcoin.node=esplora",
			"--bitcoin.esploraurl="+esploraURL.String(),
			"--bitcoin.esplorapollinterval=500ms",
		)
	} else {
		chainArgs = append(chainArgs, "--bitcoin.node=nochainbackend")
	}

	return append(chainArgs, []string{
		"--db.backend=sqlite",
		"--db.sqlite.maxconnections=1",
		"--rpclisten=127.0.0.1:10009",
		"--restlisten=127.0.0.1:8080",
		"--tor.socks=127.0.0.1:9050",
		"--tor.control=127.0.0.1:9051",
		"--nolisten",
		"--norest",
		"--no-macaroons",
		"--noseedbackup",
		"--lnddir=.",
		"--logdir=.",
		"--tlscertpath=tls.cert",
		"--tlskeypath=tls.key",
		"--debuglevel=info",
		"--logging.file.disable",
		"--maxpendingchannels=1",
		"--protocol.option-scid-alias",
		"--protocol.zero-conf",
		fmt.Sprintf("--trickledelay=%d", int64(time.Millisecond)),
	}...)
}
