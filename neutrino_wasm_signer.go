//go:build js && wasm

package lnd

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/lightningnetwork/lnd/keychain"
	"github.com/lightningnetwork/lnd/walletunlocker"
)

var wasmNeutrinoSigner = struct {
	sync.RWMutex
	pubKeyHex string
	sign      func([]byte) (string, error)
}{}

type apertureNodeSigner interface {
	SetSigner(string, func([]byte) (string, error))
}

func initNeutrinoBackendBeforeWallet(cfg *Config) bool {
	return !useNeutrinoWasmSQL(cfg)
}

func prepareWasmApertureSigners(cfg *Config,
	params *walletunlocker.WalletUnlockParams) error {

	nodeSigner, hasNodeSigner := cfg.net.(apertureNodeSigner)
	if !useNeutrinoWasmSQL(cfg) && !hasNodeSigner {
		return nil
	}
	if params == nil || params.Wallet == nil {
		return fmt.Errorf("wallet is required for wasm aperture signer")
	}
	if err := params.Wallet.Unlock(params.Password, nil); err != nil &&
		!strings.Contains(err.Error(), "wallet already unlocked") {

		return fmt.Errorf("unlock wallet for wasm aperture signer: %w",
			err)
	}

	chainKeyScope := waddrmgr.KeyScope{
		Purpose: keychain.BIP0043Purpose,
		Coin:    cfg.ActiveNetParams.CoinType,
	}
	scope, err := params.Wallet.AddrManager().FetchScopedKeyManager(
		chainKeyScope,
	)
	if waddrmgr.IsError(err, waddrmgr.ErrScopeNotFound) {
		scope, err = params.Wallet.AddScopeManager(
			chainKeyScope, waddrmgr.ScopeAddrSchema{
				ExternalAddrType: waddrmgr.WitnessPubKey,
				InternalAddrType: waddrmgr.WitnessPubKey,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("prepare wasm neutrino key scope: %w", err)
	}
	if err := params.Wallet.InitAccounts(scope, false, 255); err != nil {
		return fmt.Errorf("prepare wasm neutrino accounts: %w", err)
	}

	keyRing := keychain.NewBtcWalletKeyRing(
		params.Wallet, cfg.ActiveNetParams.CoinType,
	)
	keyLoc := keychain.KeyLocator{
		Family: keychain.KeyFamilyNodeKey,
		Index:  0,
	}
	keyDesc, err := keyRing.DeriveKey(keyLoc)
	if err != nil {
		return fmt.Errorf("derive wasm aperture node key: %w", err)
	}

	pubKeyHex := hex.EncodeToString(keyDesc.PubKey.SerializeCompressed())
	sign := func(payload []byte) (string, error) {
		sig, err := keyRing.SignMessage(keyLoc, payload, false)
		if err != nil {
			return "", err
		}

		return hex.EncodeToString(sig.Serialize()), nil
	}

	if useNeutrinoWasmSQL(cfg) {
		wasmNeutrinoSigner.Lock()
		wasmNeutrinoSigner.pubKeyHex = pubKeyHex
		wasmNeutrinoSigner.sign = sign
		wasmNeutrinoSigner.Unlock()
	}

	if hasNodeSigner {
		nodeSigner.SetSigner(pubKeyHex, sign)
	}

	return nil
}

func wasmNeutrinoClientPubKey() string {
	wasmNeutrinoSigner.RLock()
	defer wasmNeutrinoSigner.RUnlock()

	return wasmNeutrinoSigner.pubKeyHex
}

func wasmNeutrinoSignPayload(payload []byte) (string, error) {
	wasmNeutrinoSigner.RLock()
	sign := wasmNeutrinoSigner.sign
	wasmNeutrinoSigner.RUnlock()

	if sign == nil {
		return "", fmt.Errorf("wasm neutrino signer is not configured")
	}

	return sign(payload)
}
