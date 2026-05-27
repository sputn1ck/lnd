package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/lightningnetwork/lnd/brontide"
	"github.com/lightningnetwork/lnd/keychain"
	"github.com/lightningnetwork/lnd/lnwire"
)

type peerInfo struct {
	Pubkey string `json:"pubkey"`
	Host   string `json:"host"`
}

func main() {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		exitErr("create peer key", err)
	}

	listener, err := brontide.NewListener(
		&keychain.PrivKeyECDH{PrivKey: priv}, "127.0.0.1:0",
		brontide.DisabledBanClosure,
	)
	if err != nil {
		exitErr("listen", err)
	}
	defer listener.Close()

	tcpAddr := listener.Addr().(*net.TCPAddr)
	info := peerInfo{
		Pubkey: hex.EncodeToString(priv.PubKey().SerializeCompressed()),
		Host:   net.JoinHostPort("127.0.0.1", fmt.Sprint(tcpAddr.Port)),
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(info); err != nil {
		exitErr("write peer info", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			exitErr("accept", err)
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	for {
		msg, err := lnwire.ReadMessage(conn, 0)
		if err != nil {
			return
		}

		switch msg := msg.(type) {
		case *lnwire.Init:
			writeMsg(conn, lnwire.NewInitMessage(
				lnwire.NewRawFeatureVector(),
				lnwire.NewRawFeatureVector(),
			))

		case *lnwire.Ping:
			writeMsg(conn, lnwire.NewPong(make([]byte, msg.NumPongBytes)))
		}
	}
}

func writeMsg(conn net.Conn, msg lnwire.Message) {
	var buf bytes.Buffer
	if _, err := lnwire.WriteMessage(&buf, msg, 0); err != nil {
		return
	}
	_, _ = conn.Write(buf.Bytes())
}

func exitErr(action string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", action, err)
	os.Exit(1)
}
