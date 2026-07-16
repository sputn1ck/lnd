package funding

import (
	"context"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chainhash/v2"
	"github.com/btcsuite/btcd/wire/v2"
	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/fn/v2"
	"github.com/stretchr/testify/require"
)

type activationGateCall struct {
	req ChannelActivationRequest
}

type testChannelActivationGate struct {
	calls   chan activationGateCall
	release chan struct{}
}

func (g *testChannelActivationGate) WaitForActivation(ctx context.Context,
	req ChannelActivationRequest) error {

	select {
	case g.calls <- activationGateCall{req: req}:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-g.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestWaitForChannelActivation(t *testing.T) {
	t.Parallel()

	gate := &testChannelActivationGate{
		calls:   make(chan activationGateCall, 1),
		release: make(chan struct{}),
	}
	manager := &Manager{
		cfg: &Config{
			ChannelActivationGate: fn.Some[ChannelActivationGate](
				gate,
			),
		},
		quit: make(chan struct{}),
	}
	t.Cleanup(func() {
		close(manager.quit)
	})

	priv, _ := btcec.PrivKeyFromBytes([]byte{1})
	channel := &channeldb.OpenChannel{
		IdentityPub: priv.PubKey(),
		FundingOutpoint: wire.OutPoint{
			Hash: chainhash.HashH([]byte("activation-gate")),
		},
	}
	pendingID := PendingChanID{1, 2, 3}
	errChan := make(chan error, 1)
	go func() {
		errChan <- manager.waitForChannelActivation(channel, pendingID)
	}()

	select {
	case call := <-gate.calls:
		require.Equal(t, pendingID, call.req.PendingChanID)
		require.Equal(t, channel.FundingOutpoint,
			call.req.FundingOutpoint)
	case <-time.After(time.Second):
		t.Fatal("activation gate was not called")
	}

	select {
	case err := <-errChan:
		t.Fatalf("activation gate returned before release: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	close(gate.release)
	require.NoError(t, <-errChan)
}

var _ ChannelActivationGate = (*testChannelActivationGate)(nil)
