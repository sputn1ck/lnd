//go:build !js || !wasm

package lnd

import "github.com/lightninglabs/neutrino"

func useNeutrinoWasmSQL(*Config) bool {
	return false
}

func applyNeutrinoWasmOptions(*neutrino.Config, *Config) error {
	return nil
}
