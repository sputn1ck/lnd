//go:build !js || !wasm

package lnd

import "github.com/lightningnetwork/lnd/walletunlocker"

func initNeutrinoBackendBeforeWallet(*Config) bool {
	return true
}

func prepareWasmApertureSigners(*Config, *walletunlocker.WalletUnlockParams) error {
	return nil
}
