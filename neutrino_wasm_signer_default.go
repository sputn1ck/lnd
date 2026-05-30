//go:build !js || !wasm

package lnd

import "github.com/lightningnetwork/lnd/walletunlocker"

func initNeutrinoBackendBeforeWallet(*Config) bool {
	return true
}

func prepareNeutrinoWasmSigner(*Config, *walletunlocker.WalletUnlockParams) error {
	return nil
}
