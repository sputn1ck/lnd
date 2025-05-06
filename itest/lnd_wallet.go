package itest

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/davecgh/go-spew/spew"
	"github.com/lightningnetwork/lnd/input"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/signrpc"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"
	"github.com/lightningnetwork/lnd/lntest"
	"github.com/lightningnetwork/lnd/lntest/node"
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/lightningnetwork/lnd/lnwallet/chainfee"
	"github.com/stretchr/testify/require"
)

// walletTestCases defines a set of tests aiming at asserting functionalities
// provided by the wallerpc.
var walletTestCases = []*lntest.TestCase{
	{
		Name: "listunspent P2WPKH",
		TestFunc: func(ht *lntest.HarnessTest) {
			runTestListUnspent(ht, ht.FundCoins)
		},
	},
	{
		Name: "listunspent NP2WPKH",
		TestFunc: func(ht *lntest.HarnessTest) {
			runTestListUnspent(ht, ht.FundCoinsNP2WKH)
		},
	},
	{
		Name: "listunspent P2TR",
		TestFunc: func(ht *lntest.HarnessTest) {
			runTestListUnspent(ht, ht.FundCoinsP2TR)
		},
	},
	{
		Name: "listunspent P2WPKH restart",
		TestFunc: func(ht *lntest.HarnessTest) {
			runTestListUnspentRestart(ht, ht.FundCoins)
		},
	},
	{
		Name: "listunspent NP2WPKH restart",
		TestFunc: func(ht *lntest.HarnessTest) {
			runTestListUnspentRestart(ht, ht.FundCoinsNP2WKH)
		},
	},
	{
		Name: "listunspent P2TR restart",
		TestFunc: func(ht *lntest.HarnessTest) {
			runTestListUnspentRestart(ht, ht.FundCoinsP2TR)
		},
	},
	{
		Name: "submit package basic",
		TestFunc: func(ht *lntest.HarnessTest) {
			runTestSubmitPackageBasicCPFP(ht)
		},
	},
}

type fundMethod func(amt btcutil.Amount, hn *node.HarnessNode) *wire.MsgTx

// testListUnspent checks that once Alice sends all coins to Bob, her wallet
// balances are updated and the ListUnspent returns no UTXO.
func runTestListUnspent(ht *lntest.HarnessTest, fundCoins fundMethod) {
	// Create two test nodes.
	alice := ht.NewNode("Alice", nil)
	bob := ht.NewNode("Bob", nil)

	// Fund Alice one UTXO.
	coin := btcutil.Amount(100_000)
	fundCoins(coin, alice)

	// Log Alice's wallet balance for debug.
	balance := alice.RPC.WalletBalance()
	ht.Logf("Alice has balance: %v", balance)

	// Send all Alice's balance to Bob.
	ht.SendAllCoins(alice, bob)

	// Mine Alice's send coin tx.
	ht.MineBlocksAndAssertNumTxes(1, 1)

	// Alice wallet should be empty now, assert that all her balance fields
	// are zero.
	ht.AssertWalletAccountBalance(alice, lnwallet.DefaultAccountName, 0, 0)
	ht.AssertWalletLockedBalance(alice, 0)

	// Alice should have no UTXO.
	ht.AssertNumUTXOs(alice, 0)
}

// testListUnspentRestart checks that once Alice sends all coins to Bob, then
// restarts, her wallet balances are updated and the ListUnspent returns no
// UTXO.
func runTestListUnspentRestart(ht *lntest.HarnessTest, fundCoins fundMethod) {
	// Create two test nodes.
	alice := ht.NewNode("Alice", nil)
	bob := ht.NewNode("Bob", nil)

	// Fund Alice one UTXO.
	coin := btcutil.Amount(100_000)
	fundCoins(coin, alice)

	// Log Alice's wallet balance for debug.
	balance := alice.RPC.WalletBalance()
	ht.Logf("Alice has balance: %v", balance)

	// Send all Alice's balance to Bob.
	ht.SendAllCoins(alice, bob)

	// Shutdown Alice.
	restart := ht.SuspendNode(alice)

	// Mine Alice's send coin tx.
	ht.MineBlocksAndAssertNumTxes(1, 1)

	// Restart Alice.
	require.NoError(ht, restart())

	// Alice wallet should be empty now, assert that all her balance fields
	// are zero.
	ht.AssertWalletAccountBalance(alice, lnwallet.DefaultAccountName, 0, 0)
	ht.AssertWalletLockedBalance(alice, 0)

	// Alice should have no UTXO.
	ht.AssertNumUTXOs(alice, 0)
}

// Helper to convert walletrpc.KeyDescriptor to a P2WPKH address.
func keyDescToAddrP2WPKH(ht *lntest.HarnessTest,
	keyDesc *signrpc.KeyDescriptor) btcutil.Address {

	harnessNetParams := &chaincfg.RegressionNetParams

	pubKeyBytes := keyDesc.RawKeyBytes
	require.NotEmpty(ht, pubKeyBytes, "KeyDescriptor has empty RawKeyBytes")

	addr, err := btcutil.NewAddressWitnessPubKeyHash(
		btcutil.Hash160(pubKeyBytes),
		harnessNetParams,
	)
	require.NoError(ht, err)

	return addr
}

// runTestSubmitPackageBasicCPFP tests the SubmitPackage RPC for a simple
// Child-Pays-For-Parent (CPFP) scenario.
func runTestSubmitPackageBasicCPFP(ht *lntest.HarnessTest) {
	const (
		fundingAmount = btcutil.Amount(10_000_000)
		childFeeRate  = 10
		testKeyFamily = 111
	)

	// Get the chain backend RPC client.
	// irst get backend config.
	bitcoindRpc := ht.GetChainBackendRpcClient()
	// Cast to lntest.BitcoindBackendConfig.
	// btcdBackendCfg := chainBackendCfg.(*BitcoindBackendConfig)
	// btcdBackendCfg.rpcClient
	// rpcClient := ht.GetChainBackendRpcClient()
	require.NotNil(ht, bitcoindRpc, "Failed to get chain backend RPC client")

	// Create a new node (Alice).
	alice := ht.NewNode("Alice", nil)
	defer ht.Shutdown(alice)

	// Fund Alice with a UTXO.
	ht.FundCoins(fundingAmount*2, alice)

	// Derive a key and address for the initial funding.
	aliceKey := alice.RPC.DeriveNextKey(
		&walletrpc.KeyReq{
			KeyFamily: testKeyFamily,
		},
	)
	aliceAddr := keyDescToAddrP2WPKH(ht, aliceKey)

	ht.Logf("Funding address: %s", aliceAddr)

	// Fund this specific address.
	ht.SendCoinsToAddr(
		alice, aliceAddr, fundingAmount,
	)
	block := ht.MineBlocksAndAssertNumTxes(1, 1)

	expectedScript := ht.PayToAddrScript(aliceAddr)
	var (
		outputIndex   uint32
		parentPrevOut *wire.TxOut
		found         bool
		fundTx        *wire.MsgTx
	)
	for _, tx := range block[0].Transactions {
		for i, txOut := range tx.TxOut {
			if bytes.Equal(txOut.PkScript, expectedScript) {
				outputIndex = uint32(i)
				parentPrevOut = txOut
				found = true
				fundTx = tx
				break
			}
		}
	}
	require.True(ht, found, "funding output not found")

	fundOutPoint := wire.OutPoint{
		Hash:  fundTx.TxHash(),
		Index: outputIndex,
	}
	// Derive a key and address for the parent's output.
	parentOutKeyDesc := alice.RPC.DeriveNextKey(
		&walletrpc.KeyReq{
			KeyFamily: testKeyFamily,
		},
	)
	parentOutAddr := keyDescToAddrP2WPKH(ht, parentOutKeyDesc)
	parentOutPkScript := ht.PayToAddrScript(parentOutAddr)

	parentOutputValue := btcutil.Amount(fundingAmount)

	parentTx := wire.NewMsgTx(3)
	parentTx.AddTxIn(wire.NewTxIn(&fundOutPoint, nil, nil))
	parentTx.AddTxOut(
		wire.NewTxOut(int64(parentOutputValue), parentOutPkScript),
	)

	// Serialize the parent tx.
	var parentBuf bytes.Buffer
	require.NoError(ht, parentTx.Serialize(&parentBuf))

	// Create a sign descriptor for the parent tx.
	parentSignDesc := &signrpc.SignDescriptor{
		KeyDesc: aliceKey,
		Output: &signrpc.TxOut{
			Value:    parentPrevOut.Value,
			PkScript: parentPrevOut.PkScript,
		},
		Sighash:       uint32(txscript.SigHashAll),
		WitnessScript: parentPrevOut.PkScript,
		SignMethod:    signrpc.SignMethod_SIGN_METHOD_WITNESS_V0,
	}
	parentSignResp, err := alice.RPC.Signer.SignOutputRaw(
		ht.Context(),
		&signrpc.SignReq{
			RawTxBytes: parentBuf.Bytes(),
			SignDescs:  []*signrpc.SignDescriptor{parentSignDesc},
		},
	)
	require.NoError(ht, err)

	// Set the parent tx's witness to the signature.
	parentTx.TxIn[0].Witness = makeP2WPKHWitness(
		parentSignResp.RawSigs[0],
		aliceKey,
	)

	// Create a new P2WPKH address for the child's output.
	childOutAddrResp := alice.RPC.NewAddress(
		&lnrpc.NewAddressRequest{
			Account: lnwallet.DefaultAccountName,
			Type:    lnrpc.AddressType_WITNESS_PUBKEY_HASH,
		},
	)
	require.NoError(ht, err)

	childOutPkScript := ht.PayToAddrScript(
		ht.DecodeAddress(childOutAddrResp.Address),
	)

	// Estimate child's weight and fee.
	childEstimator := input.TxWeightEstimator{}
	childEstimator.AddP2WKHInput()
	childEstimator.AddP2WKHOutput()
	childWeight := childEstimator.Weight()
	childFee := chainfee.SatPerVByte(
		childFeeRate,
	).FeePerKWeight().FeeForWeight(
		childWeight,
	)

	childOutputValue := btcutil.Amount(parentOutputValue) - childFee

	// Create the unsigned child transaction.
	childTx := wire.NewMsgTx(3)
	childParentInputOP := wire.OutPoint{
		Hash:  parentTx.TxHash(),
		Index: 0,
	}

	childTx.AddTxIn(wire.NewTxIn(&childParentInputOP, nil, nil))
	childTx.AddTxOut(
		wire.NewTxOut(int64(childOutputValue), childOutPkScript),
	)
	var childBuf bytes.Buffer
	require.NoError(ht, childTx.Serialize(&childBuf))

	childPrevOut := &wire.TxOut{
		Value:    int64(parentOutputValue),
		PkScript: parentOutPkScript, // script of *parent’s* output
	}

	childSignDesc := &signrpc.SignDescriptor{
		KeyDesc: parentOutKeyDesc,
		Output: &signrpc.TxOut{
			Value:    childPrevOut.Value,
			PkScript: childPrevOut.PkScript,
		},
		WitnessScript: childPrevOut.PkScript,
		Sighash:       uint32(txscript.SigHashAll),
		SignMethod:    signrpc.SignMethod_SIGN_METHOD_WITNESS_V0,
	}

	childSignResp, err := alice.RPC.Signer.SignOutputRaw(
		ht.Context(),
		&signrpc.SignReq{
			RawTxBytes: childBuf.Bytes(),
			SignDescs:  []*signrpc.SignDescriptor{childSignDesc},
		},
	)
	require.NoError(ht, err)

	// Set the child tx's witness to the signature.
	childTx.TxIn[0].Witness = makeP2WPKHWitness(
		childSignResp.RawSigs[0],
		parentOutKeyDesc,
	)

	// Reserialize parent and child txns as they now have signatures.
	parentBuf = bytes.Buffer{}
	require.NoError(ht, parentTx.Serialize(&parentBuf))
	parentHex := hex.EncodeToString(parentBuf.Bytes())
	childBuf = bytes.Buffer{}
	require.NoError(ht, childTx.Serialize(&childBuf))
	childHex := hex.EncodeToString(childBuf.Bytes())
	decodedParentTx, err := bitcoindRpc.DecodeRawTransaction(parentBuf.Bytes())
	require.NoError(ht, err)
	decodedChildTx, err := bitcoindRpc.DecodeRawTransaction(childBuf.Bytes())
	require.NoError(ht, err)
	ht.Logf("Decoded parent tx: %v", spew.Sdump(decodedParentTx))
	ht.Logf("Decoded child tx: %v", spew.Sdump(decodedChildTx))
	rawTxs := []string{
		parentHex,
		childHex,
	}

	rawTxsJSON, err := json.Marshal(rawTxs)
	require.NoError(ht, err)

	info, err := ht.Miner().Harness.Client.GetInfo()
	require.NoError(ht, err)
	ht.Logf("Miner info: %v", spew.Sdump(info))

	resp, err := bitcoindRpc.RawRequest(
		"submitpackage",
		[]json.RawMessage{
			rawTxsJSON,
		},
	)
	require.NoError(ht, err)

	// Unmarshall the response.
	var result btcjson.SubmitPackageResult
	err = json.Unmarshal(resp, &result)
	require.NoError(ht, err)
	ht.Logf("SubmitPackage response: %v", spew.Sdump(result))
	for _, txid := range result.TxResults {
		require.Empty(ht, txid.Error, "Expected no error in response")
	}

	ht.AssertTxInMempool(parentTx.TxHash())
	ht.AssertTxInMempool(childTx.TxHash())

	ht.Log("Mining block with package transactions...")
	block = ht.MineBlocksAndAssertNumTxes(1, 2)
	ht.AssertTxInBlock(block[0], parentTx.TxHash())
	ht.AssertTxInBlock(block[0], childTx.TxHash())
}

func makeP2WPKHWitness(sig []byte, keyDesc *signrpc.KeyDescriptor) wire.TxWitness {
	sigWithHash := append(sig, byte(txscript.SigHashAll))
	return wire.TxWitness{
		sigWithHash,         // element 0: 70-73-byte signature
		keyDesc.RawKeyBytes, // element 1: 33-byte compressed pubkey
	}
}
