package lnwallet

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// lifecycleWalletController records whether LightningWallet takes ownership
// of its embedded wallet controller. The nil embedded interface is safe because
// this test only exercises Start and Stop.
type lifecycleWalletController struct {
	WalletController

	starts atomic.Int32
	stops  atomic.Int32
}

// Start records a wallet-controller start.
func (c *lifecycleWalletController) Start() error {
	c.starts.Add(1)

	return nil
}

// Stop records a wallet-controller stop.
func (c *lifecycleWalletController) Stop() error {
	c.stops.Add(1)

	return nil
}

// TestRegisterFundingIntent checks RegisterFundingIntent behaves as expected.
func TestRegisterFundingIntent(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	// Create a testing wallet.
	lw, err := NewLightningWallet(Config{})
	require.NoError(err)

	// Init an empty testing channel ID.
	var testID [32]byte

	// Call the method with empty ID should give us an error.
	err = lw.RegisterFundingIntent(testID, nil)
	require.ErrorIs(err, ErrEmptyPendingChanID)

	// Modify the ID and call the method again should result in no error.
	testID[0] = 1
	err = lw.RegisterFundingIntent(testID, nil)
	require.NoError(err)

	// Call the method using the same ID should give us an error.
	err = lw.RegisterFundingIntent(testID, nil)
	require.ErrorIs(err, ErrDuplicatePendingChanID)
}

// TestExternallyManagedWalletController verifies an embedding process can run
// LightningWallet's reservation state machine without transferring ownership
// of its existing wallet lifecycle.
func TestExternallyManagedWalletController(t *testing.T) {
	t.Parallel()

	controller := &lifecycleWalletController{}
	wallet, err := NewLightningWallet(Config{
		WalletController:                  controller,
		ExternallyManagedWalletController: true,
	})
	require.NoError(t, err)
	require.NoError(t, wallet.Startup())
	require.NoError(t, wallet.Shutdown())
	require.Zero(t, controller.starts.Load())
	require.Zero(t, controller.stops.Load())
}

// TestManagedWalletController verifies the default lnd lifecycle remains
// unchanged.
func TestManagedWalletController(t *testing.T) {
	t.Parallel()

	controller := &lifecycleWalletController{}
	wallet, err := NewLightningWallet(Config{
		WalletController: controller,
	})
	require.NoError(t, err)
	require.NoError(t, wallet.Startup())
	require.NoError(t, wallet.Shutdown())
	require.EqualValues(t, 1, controller.starts.Load())
	require.EqualValues(t, 1, controller.stops.Load())
}
