//go:build js && wasm

package lnd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall/js"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/lightninglabs/neutrino"
	neutrinosql "github.com/lightninglabs/neutrino/sqldb"
	"github.com/lightninglabs/neutrino/wasmtransport"
	sqldbv2 "github.com/lightningnetwork/lnd/sqldb/v2"
)

const (
	defaultNeutrinoWasmProxyURL = "wss://konwss.tunn.dev/peer-proxy"
	defaultNeutrinoWasmDNSURL   = "https://cloudflare-dns.com/dns-query"
)

func useNeutrinoWasmSQL(cfg *Config) bool {
	return cfg.Bitcoin.Node == "neutrino"
}

func applyNeutrinoWasmOptions(config *neutrino.Config, cfg *Config) error {
	if !useNeutrinoWasmSQL(cfg) {
		return nil
	}

	neutrino.DisableDNSSeed = false

	config.DataDir = "/"
	config.Database = nil
	config.SQLConfig = &neutrinosql.Config{
		Backend: neutrinosql.BackendSqlite,
		Sqlite: &sqldbv2.SqliteConfig{
			BusyTimeout: 5 * time.Second,
		},
		SqliteFilename:      "neutrino.sqlite",
		SkipLegacyMigration: true,
	}

	proxyURL := wasmGlobalString("lndWasmNeutrinoProxyURL")
	if proxyURL == "" {
		proxyURL = defaultNeutrinoWasmProxyURL
	}
	proxyURL = normalizeWasmProxyURL(proxyURL)

	dnsURL := wasmGlobalString("lndWasmNeutrinoDNSURL")
	if dnsURL == "" {
		dnsURL = defaultNeutrinoWasmDNSURL
	}

	config.NameResolver = wasmDNSResolver(dnsURL)
	config.Dialer = wasmtransport.NewProxyDialer(proxyURL)
	config.AddrResolver = func(addr string) (net.Addr, error) {
		if isWasmOnionTarget(addr) {
			return nil, fmt.Errorf("onion neutrino peers are not " +
				"reachable from browser wasm")
		}

		addr = withWasmDefaultPort(
			addr, cfg.ActiveNetParams.Params.DefaultPort,
		)

		return wasmtransport.NewAddr(addr), nil
	}
	if config.HeadersImport != nil {
		config.HeadersImport.ValidationFlags = blockchain.BFFastAdd
	}

	return nil
}

func wasmDNSResolver(dnsURL string) func(string) ([]net.IP, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	return func(host string) ([]net.IP, error) {
		if ip := net.ParseIP(host); ip != nil {
			return []net.IP{ip}, nil
		}

		var result []net.IP
		for _, recordType := range []string{"A", "AAAA"} {
			ips, err := wasmDNSLookup(client, dnsURL, host, recordType)
			if err != nil {
				return nil, err
			}

			result = append(result, ips...)
		}

		if len(result) == 0 {
			return nil, fmt.Errorf("no DNS answers for %s", host)
		}

		return result, nil
	}
}

func wasmDNSLookup(client *http.Client, dnsURL, host,
	recordType string) ([]net.IP, error) {

	u, err := url.Parse(dnsURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("name", host)
	q.Set("type", recordType)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/dns-json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dns lookup %s %s failed: %s",
			host, recordType, resp.Status)
	}

	var dnsResp struct {
		Answer []struct {
			Type int    `json:"type"`
			Data string `json:"data"`
		} `json:"Answer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dnsResp); err != nil {
		return nil, err
	}

	wantType := 1
	if recordType == "AAAA" {
		wantType = 28
	}

	var ips []net.IP
	for _, answer := range dnsResp.Answer {
		if answer.Type != wantType {
			continue
		}

		ip := net.ParseIP(answer.Data)
		if ip == nil {
			continue
		}

		ips = append(ips, ip)
	}

	return ips, nil
}

func wasmGlobalString(name string) string {
	value := js.Global().Get(name)
	if value.Type() != js.TypeString {
		return ""
	}

	return strings.TrimSpace(value.String())
}

func normalizeWasmProxyURL(proxyURL string) string {
	return strings.ReplaceAll(strings.TrimSpace(proxyURL), ",", ".")
}

func withWasmDefaultPort(addr, defaultPort string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" || defaultPort == "" {
		return addr
	}

	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}

	host := strings.TrimPrefix(strings.TrimSuffix(addr, "]"), "[")
	return net.JoinHostPort(host, defaultPort)
}

func isWasmOnionTarget(addr string) bool {
	host := addr
	if strings.HasPrefix(host, "[") {
		if end := strings.Index(host, "]"); end >= 0 {
			host = host[1:end]
		}
	} else if before, _, ok := strings.Cut(host, ":"); ok {
		host = before
	}

	return strings.HasSuffix(strings.ToLower(host), ".onion")
}
