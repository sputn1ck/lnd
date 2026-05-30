//go:build js && wasm

package lnd

import (
	"strings"
	"syscall/js"

	"github.com/lightninglabs/ll-wasm/go/wasmneutrino"
	"github.com/lightninglabs/neutrino"
)

func useNeutrinoWasmSQL(cfg *Config) bool {
	return cfg.Bitcoin.Node == "neutrino"
}

func applyNeutrinoWasmOptions(config *neutrino.Config, cfg *Config) error {
	if !useNeutrinoWasmSQL(cfg) {
		return nil
	}

	dnsURL := wasmGlobalString("lndWasmNeutrinoDNSURL")
	if dnsURL == "" {
		dnsURL = wasmneutrino.DefaultDoHEndpoint
	}

	apertureProxyURL := wasmGlobalString("lndWasmApertureProxyURL")
	return wasmneutrino.ApplyBrowserOptions(config, wasmneutrino.BrowserConfig{
		DataDir:              "/",
		DBFilename:           "neutrino.sqlite",
		DNSURL:               dnsURL,
		ChainParams:          cfg.ActiveNetParams.Params,
		SQLiteVFS:            wasmGlobalString("lndWasmSQLiteVFS"),
		ApertureProxyURL:     apertureProxyURL,
		ApertureClientPubKey: wasmNeutrinoClientPubKey,
		ApertureSignPayload:  wasmNeutrinoSignPayload,
	})
}

func wasmGlobalString(name string) string {
	value := js.Global().Get(name)
	if value.Type() != js.TypeString {
		return ""
	}

	return strings.TrimSpace(value.String())
}
