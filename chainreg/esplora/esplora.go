package esplora

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btclog/v2"
	walletchain "github.com/btcsuite/btcwallet/chain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/wtxmgr"
	"github.com/lightningnetwork/lnd/chainntnfs"
	graphdb "github.com/lightningnetwork/lnd/graph/db"
	"github.com/lightningnetwork/lnd/lnwallet/btcwallet"
	"github.com/lightningnetwork/lnd/lnwallet/chainfee"
	"github.com/lightningnetwork/lnd/routing/chainview"
)

const (
	DefaultPollInterval = 2 * time.Second
	backendName         = "esplora"
	maxRequestAttempts  = 5
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    newHTTPClient(),
	}
}

type blockMeta struct {
	ID        string `json:"id"`
	Height    int32  `json:"height"`
	Timestamp int64  `json:"timestamp"`
}

type txStatus struct {
	Confirmed   bool   `json:"confirmed"`
	BlockHeight uint32 `json:"block_height"`
	BlockHash   string `json:"block_hash"`
}

type utxo struct {
	Txid   string   `json:"txid"`
	Vout   uint32   `json:"vout"`
	Value  int64    `json:"value"`
	Status txStatus `json:"status"`
}

type outspend struct {
	Spent  bool     `json:"spent"`
	Txid   string   `json:"txid"`
	Vin    uint32   `json:"vin"`
	Status txStatus `json:"status"`
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, http.MethodGet, path, "", "")
}

func (c *Client) post(ctx context.Context, path string, body string,
	contentType string) ([]byte, error) {

	return c.request(ctx, http.MethodPost, path, body, contentType)
}

func (c *Client) request(ctx context.Context, method, path, body,
	contentType string) ([]byte, error) {

	var lastErr error
	for attempt := 0; attempt < maxRequestAttempts; attempt++ {
		respBody, retryAfter, retry, err := c.requestOnce(
			ctx, method, path, body, contentType,
		)
		if err == nil {
			return respBody, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}

		wait := retryDelay(attempt, retryAfter)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-time.After(wait):
		}
	}

	return nil, fmt.Errorf("esplora %s %s failed after %d attempts: %w",
		method, path, maxRequestAttempts, lastErr)
}

func (c *Client) requestOnce(ctx context.Context, method, path, body,
	contentType string) ([]byte, time.Duration, bool, error) {

	reqBody := strings.NewReader(body)
	req, err := http.NewRequestWithContext(
		ctx, method, c.baseURL+path, reqBody,
	)
	if err != nil {
		return nil, 0, false, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req) //nolint:gosec // User configured backend.
	if err != nil {
		return nil, 0, true, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, true, err
	}
	if resp.StatusCode != http.StatusOK {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		retry := retryStatus(resp.StatusCode)
		return nil, retryAfter, retry, fmt.Errorf(
			"esplora %s %s: HTTP %d: %s", method, path,
			resp.StatusCode, string(respBody),
		)
	}

	return respBody, 0, false, nil
}

func retryStatus(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusRequestTimeout ||
		code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout ||
		code >= 500
}

func retryDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}

	delay := time.Duration(250*(1<<attempt)) * time.Millisecond
	if delay > 4*time.Second {
		return 4 * time.Second
	}

	return delay
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		delay := time.Until(when)
		if delay > 0 {
			return delay
		}
	}

	return 0
}

func (c *Client) TipHeight(ctx context.Context) (int32, error) {
	body, err := c.get(ctx, "/blocks/tip/height")
	if err != nil {
		return 0, err
	}

	height, err := strconv.ParseInt(strings.TrimSpace(string(body)), 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(height), nil
}

func (c *Client) BlockHashByHeight(ctx context.Context,
	height int32) (chainhash.Hash, error) {

	body, err := c.get(ctx, "/block-height/"+strconv.Itoa(int(height)))
	if err != nil {
		return chainhash.Hash{}, err
	}

	hash, err := chainhash.NewHashFromStr(strings.TrimSpace(string(body)))
	if err != nil {
		return chainhash.Hash{}, err
	}

	return *hash, nil
}

func (c *Client) BlockMeta(ctx context.Context,
	hash chainhash.Hash) (*blockMeta, error) {

	body, err := c.get(ctx, "/block/"+hash.String())
	if err != nil {
		return nil, err
	}

	var meta blockMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, err
	}

	got, err := chainhash.NewHashFromStr(meta.ID)
	if err != nil {
		return nil, err
	}
	if *got != hash {
		return nil, fmt.Errorf("block id mismatch: got %s want %s",
			got, hash)
	}

	return &meta, nil
}

func (c *Client) RawBlockHeader(ctx context.Context,
	hash chainhash.Hash) (*wire.BlockHeader, error) {

	body, err := c.get(ctx, "/block/"+hash.String()+"/header")
	if err != nil {
		return nil, err
	}

	headerBytes, err := hex.DecodeString(strings.TrimSpace(string(body)))
	if err != nil {
		return nil, err
	}

	var header wire.BlockHeader
	if err := header.Deserialize(bytes.NewReader(headerBytes)); err != nil {
		return nil, err
	}
	if got := header.BlockHash(); got != hash {
		return nil, fmt.Errorf("header hash mismatch: got %s want %s",
			got, hash)
	}

	return &header, nil
}

func (c *Client) RawBlock(ctx context.Context,
	hash chainhash.Hash) (*wire.MsgBlock, error) {

	body, err := c.get(ctx, "/block/"+hash.String()+"/raw")
	if err != nil {
		return nil, err
	}

	var block wire.MsgBlock
	if err := block.Deserialize(bytes.NewReader(body)); err != nil {
		return nil, err
	}
	if got := block.BlockHash(); got != hash {
		return nil, fmt.Errorf("block hash mismatch: got %s want %s",
			got, hash)
	}

	return &block, nil
}

func (c *Client) RawTx(ctx context.Context,
	txid chainhash.Hash) (*wire.MsgTx, error) {

	body, err := c.get(ctx, "/tx/"+txid.String()+"/raw")
	if err != nil {
		return nil, err
	}

	var tx wire.MsgTx
	if err := tx.Deserialize(bytes.NewReader(body)); err != nil {
		return nil, err
	}
	if got := tx.TxHash(); got != txid {
		return nil, fmt.Errorf("tx hash mismatch: got %s want %s",
			got, txid)
	}

	return &tx, nil
}

func (c *Client) TxStatus(ctx context.Context,
	txid chainhash.Hash) (*txStatus, error) {

	body, err := c.get(ctx, "/tx/"+txid.String()+"/status")
	if err != nil {
		return nil, err
	}

	var status txStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

func (c *Client) ScriptUTXOs(ctx context.Context, pkScript []byte) ([]utxo,
	error) {

	h := sha256.Sum256(pkScript)
	for i, j := 0, len(h)-1; i < j; i, j = i+1, j-1 {
		h[i], h[j] = h[j], h[i]
	}

	body, err := c.get(ctx, "/scripthash/"+hex.EncodeToString(h[:])+"/utxo")
	if err != nil {
		return nil, err
	}

	var utxos []utxo
	if err := json.Unmarshal(body, &utxos); err != nil {
		return nil, err
	}

	return utxos, nil
}

func (c *Client) Outspend(ctx context.Context, op wire.OutPoint) (*outspend,
	error) {

	path := fmt.Sprintf("/tx/%s/outspend/%d", op.Hash, op.Index)
	body, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	var spend outspend
	if err := json.Unmarshal(body, &spend); err != nil {
		return nil, err
	}

	return &spend, nil
}

func (c *Client) FeeEstimates(ctx context.Context) (map[string]float64,
	error) {

	body, err := c.get(ctx, "/fee-estimates")
	if err != nil {
		return nil, err
	}

	var estimates map[string]float64
	if err := json.Unmarshal(body, &estimates); err != nil {
		return nil, err
	}

	return estimates, nil
}

func (c *Client) BroadcastTx(ctx context.Context, tx *wire.MsgTx) (
	*chainhash.Hash, error) {

	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return nil, err
	}
	txHex := hex.EncodeToString(buf.Bytes())

	_, err := c.post(
		ctx, "/tx", txHex, "text/plain",
	)
	if err != nil {
		fallbackURL := broadcastFallbackURL(c.baseURL)
		if fallbackURL == "" || !missingEndpoint(err) {
			return nil, err
		}

		fallback := &Client{
			baseURL: strings.TrimRight(fallbackURL, "/"),
			http:    c.http,
		}
		_, fallbackErr := fallback.post(
			ctx, "/tx", txHex, "text/plain",
		)
		if fallbackErr != nil {
			return nil, fmt.Errorf("%w; fallback broadcast via "+
				"%s failed: %v", err, fallbackURL, fallbackErr)
		}
	}

	txid := tx.TxHash()
	return &txid, nil
}

func missingEndpoint(err error) bool {
	return strings.Contains(err.Error(), "endpoint does not exist")
}

func (c *Client) TestMempoolAccept(ctx context.Context, txns []*wire.MsgTx,
	maxFeeRate float64) ([]*btcjson.TestMempoolAcceptResult, error) {

	txHexes := make([]string, 0, len(txns))
	for _, tx := range txns {
		var buf bytes.Buffer
		if err := tx.Serialize(&buf); err != nil {
			return nil, err
		}
		txHexes = append(txHexes, hex.EncodeToString(buf.Bytes()))
	}

	body, err := json.Marshal(txHexes)
	if err != nil {
		return nil, err
	}

	path := "/txs/test"
	if maxFeeRate > 0 {
		path += fmt.Sprintf("?maxfeerate=%f", maxFeeRate)
	}

	// Some public Esplora deployments accept the JSON payload but don't
	// answer CORS preflight requests for this endpoint. text/plain keeps the
	// browser request simple while preserving the same JSON body.
	resp, err := c.post(ctx, path, string(body), "text/plain")
	if err != nil {
		return nil, err
	}

	var results []*btcjson.TestMempoolAcceptResult
	if err := json.Unmarshal(resp, &results); err != nil {
		return nil, err
	}

	return results, nil
}

type tipBlock struct {
	height int32
	hash   chainhash.Hash
	header *wire.BlockHeader
}

type confReg struct {
	txid         *chainhash.Hash
	pkScript     []byte
	numConfs     uint32
	includeBlock bool
	event        *chainntnfs.ConfirmationEvent
}

type spendReg struct {
	outpoint *wire.OutPoint
	event    *chainntnfs.SpendEvent
}

type blockReg struct {
	epochs chan *chainntnfs.BlockEpoch
}

type Core struct {
	client       *Client
	pollInterval time.Duration
	log          btclog.Logger

	mu          sync.Mutex
	started     bool
	stopped     bool
	bestHeight  int32
	bestHash    chainhash.Hash
	bestHeader  *wire.BlockHeader
	nextRegID   uint64
	confRegs    map[uint64]*confReg
	spendRegs   map[uint64]*spendReg
	blockRegs   map[uint64]*blockReg
	watchedAddr map[string]btcutil.Address
	filterOps   map[wire.OutPoint]struct{}

	notifications chan interface{}
	filtered      chan *chainview.FilteredBlock
	disconnected  chan *chainview.FilteredBlock
	quit          chan struct{}
	stopOnce      sync.Once
	wg            sync.WaitGroup
}

func NewCore(client *Client, pollInterval time.Duration,
	logger btclog.Logger) *Core {

	if pollInterval == 0 {
		pollInterval = DefaultPollInterval
	}
	if logger == nil {
		logger = btclog.Disabled
	}

	return &Core{
		client:        client,
		pollInterval:  pollInterval,
		log:           logger,
		confRegs:      make(map[uint64]*confReg),
		spendRegs:     make(map[uint64]*spendReg),
		blockRegs:     make(map[uint64]*blockReg),
		watchedAddr:   make(map[string]btcutil.Address),
		filterOps:     make(map[wire.OutPoint]struct{}),
		notifications: make(chan interface{}, 100),
		filtered:      make(chan *chainview.FilteredBlock, 100),
		disconnected:  make(chan *chainview.FilteredBlock, 1),
		quit:          make(chan struct{}),
	}
}

func (c *Core) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	if c.stopped {
		c.mu.Unlock()
		return fmt.Errorf("esplora backend already stopped")
	}
	c.started = true
	c.mu.Unlock()

	height, err := c.client.TipHeight(ctx)
	if err != nil {
		return fmt.Errorf("get initial tip height: %w", err)
	}
	hash, err := c.client.BlockHashByHeight(ctx, height)
	if err != nil {
		return fmt.Errorf("get initial tip hash: %w", err)
	}
	header, err := c.client.RawBlockHeader(ctx, hash)
	if err != nil {
		return fmt.Errorf("get initial tip header: %w", err)
	}

	c.mu.Lock()
	c.bestHeight = height
	c.bestHash = hash
	c.bestHeader = header
	c.mu.Unlock()

	select {
	case c.notifications <- walletchain.ClientConnected{}:
	default:
	}

	c.wg.Add(1)
	go c.pollLoop()

	return nil
}

func (c *Core) Stop() error {
	c.stopOnce.Do(func() {
		c.mu.Lock()
		c.stopped = true
		c.mu.Unlock()

		close(c.quit)
	})

	c.wg.Wait()
	return nil
}

func (c *Core) Started() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.started && !c.stopped
}

func (c *Core) pollLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.poll()

		case <-c.quit:
			return
		}
	}
}

func (c *Core) poll() {
	ctx := context.Background()

	newHeight, err := c.client.TipHeight(ctx)
	if err != nil {
		c.log.Warnf("Unable to poll Esplora tip height: %v", err)
		return
	}

	c.mu.Lock()
	oldHeight := c.bestHeight
	c.mu.Unlock()

	if newHeight <= oldHeight {
		c.recheckRegistrations()
		return
	}

	for height := oldHeight + 1; height <= newHeight; height++ {
		hash, err := c.client.BlockHashByHeight(ctx, height)
		if err != nil {
			c.log.Warnf("Unable to fetch Esplora block hash: %v", err)
			return
		}
		header, err := c.client.RawBlockHeader(ctx, hash)
		if err != nil {
			c.log.Warnf("Unable to fetch Esplora block header: %v", err)
			return
		}

		c.processTip(&tipBlock{
			height: height,
			hash:   hash,
			header: header,
		})
	}
}

func (c *Core) processTip(tip *tipBlock) {
	c.mu.Lock()
	c.bestHeight = tip.height
	c.bestHash = tip.hash
	c.bestHeader = tip.header

	blockRegs := make([]*blockReg, 0, len(c.blockRegs))
	for _, reg := range c.blockRegs {
		blockRegs = append(blockRegs, reg)
	}
	watched := make(map[string]btcutil.Address, len(c.watchedAddr))
	for k, v := range c.watchedAddr {
		watched[k] = v
	}
	hasFilter := len(c.filterOps) > 0
	c.mu.Unlock()

	epoch := &chainntnfs.BlockEpoch{
		Hash:        &tip.hash,
		Height:      tip.height,
		BlockHeader: tip.header,
	}
	for _, reg := range blockRegs {
		select {
		case reg.epochs <- epoch:
		default:
		}
	}

	c.notifyWalletBlock(tip, watched)

	if hasFilter {
		filtered, err := c.filterBlock(context.Background(), tip.hash)
		if err == nil && len(filtered.Transactions) > 0 {
			select {
			case c.filtered <- filtered:
			default:
			}
		}
	}

	c.recheckRegistrations()
}

func (c *Core) notifyWalletBlock(tip *tipBlock,
	watched map[string]btcutil.Address) {

	meta := wtxmgr.BlockMeta{
		Block: wtxmgr.Block{
			Hash:   tip.hash,
			Height: tip.height,
		},
		Time: tip.header.Timestamp,
	}

	var relevant []*wtxmgr.TxRecord
	if len(watched) > 0 {
		scripts := make(map[string]struct{}, len(watched))
		for _, addr := range watched {
			pkScript, err := txscript.PayToAddrScript(addr)
			if err == nil {
				scripts[string(pkScript)] = struct{}{}
			}
		}

		block, err := c.client.RawBlock(context.Background(), tip.hash)
		if err == nil {
			relevant = filterWalletTxs(block, scripts, tip.header.Timestamp)
		}
	}

	select {
	case c.notifications <- walletchain.FilteredBlockConnected{
		Block:       &meta,
		RelevantTxs: relevant,
	}:
	case <-c.quit:
		return
	}

	select {
	case c.notifications <- walletchain.BlockConnected(meta):
	case <-c.quit:
		return
	}
}

func filterWalletTxs(block *wire.MsgBlock, scripts map[string]struct{},
	blockTime time.Time) []*wtxmgr.TxRecord {

	var records []*wtxmgr.TxRecord
	for _, tx := range block.Transactions {
		matches := false
		for _, out := range tx.TxOut {
			if _, ok := scripts[string(out.PkScript)]; ok {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}

		record, err := wtxmgr.NewTxRecordFromMsgTx(tx, blockTime)
		if err == nil {
			records = append(records, record)
		}
	}

	return records
}

func (c *Core) recheckRegistrations() {
	c.mu.Lock()
	height := c.bestHeight
	confRegs := make(map[uint64]*confReg, len(c.confRegs))
	for id, reg := range c.confRegs {
		confRegs[id] = reg
	}
	spendRegs := make(map[uint64]*spendReg, len(c.spendRegs))
	for id, reg := range c.spendRegs {
		spendRegs[id] = reg
	}
	c.mu.Unlock()

	for id, reg := range confRegs {
		conf := c.checkConf(reg, height)
		if conf == nil {
			continue
		}

		select {
		case reg.event.Confirmed <- conf:
		default:
		}
		select {
		case reg.event.Done <- struct{}{}:
		default:
		}

		c.mu.Lock()
		delete(c.confRegs, id)
		c.mu.Unlock()
	}

	for id, reg := range spendRegs {
		detail := c.checkSpend(reg)
		if detail == nil {
			continue
		}

		select {
		case reg.event.Spend <- detail:
		default:
		}
		select {
		case reg.event.Done <- struct{}{}:
		default:
		}

		c.mu.Lock()
		delete(c.spendRegs, id)
		c.mu.Unlock()
	}
}

func (c *Core) checkConf(reg *confReg,
	currentHeight int32) *chainntnfs.TxConfirmation {

	if reg.numConfs == 0 {
		reg.numConfs = 1
	}

	if reg.txid != nil {
		return c.checkConfByTxID(reg, currentHeight)
	}

	utxos, err := c.client.ScriptUTXOs(context.Background(), reg.pkScript)
	if err != nil {
		return nil
	}
	for _, u := range utxos {
		if !u.Status.Confirmed {
			continue
		}
		confHeight := int32(u.Status.BlockHeight)
		if currentHeight-confHeight+1 < int32(reg.numConfs) {
			continue
		}

		txid, err := chainhash.NewHashFromStr(u.Txid)
		if err != nil {
			continue
		}
		return c.confDetails(reg, *txid, u.Status)
	}

	return nil
}

func (c *Core) checkConfByTxID(reg *confReg,
	currentHeight int32) *chainntnfs.TxConfirmation {

	status, err := c.client.TxStatus(context.Background(), *reg.txid)
	if err != nil || !status.Confirmed {
		return nil
	}

	confHeight := int32(status.BlockHeight)
	if currentHeight-confHeight+1 < int32(reg.numConfs) {
		left := uint32(int32(reg.numConfs) - (currentHeight - confHeight + 1))
		select {
		case reg.event.Updates <- chainntnfs.TxUpdateInfo{
			BlockHeight:  status.BlockHeight,
			NumConfsLeft: left,
		}:
		default:
		}

		return nil
	}

	return c.confDetails(reg, *reg.txid, *status)
}

func (c *Core) confDetails(reg *confReg, txid chainhash.Hash,
	status txStatus) *chainntnfs.TxConfirmation {

	blockHash, err := chainhash.NewHashFromStr(status.BlockHash)
	if err != nil {
		return nil
	}
	tx, err := c.client.RawTx(context.Background(), txid)
	if err != nil {
		return nil
	}

	conf := &chainntnfs.TxConfirmation{
		BlockHash:   blockHash,
		BlockHeight: status.BlockHeight,
		Tx:          tx,
	}

	if reg.includeBlock {
		block, err := c.client.RawBlock(context.Background(), *blockHash)
		if err == nil {
			conf.Block = block
			for idx, blockTx := range block.Transactions {
				if blockTx.TxHash() == txid {
					conf.TxIndex = uint32(idx)
					break
				}
			}
		}
	}

	return conf
}

func (c *Core) checkSpend(reg *spendReg) *chainntnfs.SpendDetail {
	if reg.outpoint == nil {
		return nil
	}

	spend, err := c.client.Outspend(context.Background(), *reg.outpoint)
	if err != nil || !spend.Spent || !spend.Status.Confirmed {
		return nil
	}

	spenderHash, err := chainhash.NewHashFromStr(spend.Txid)
	if err != nil {
		return nil
	}
	spendingTx, err := c.client.RawTx(context.Background(), *spenderHash)
	if err != nil {
		return nil
	}

	return &chainntnfs.SpendDetail{
		SpentOutPoint:     reg.outpoint,
		SpenderTxHash:     spenderHash,
		SpendingTx:        spendingTx,
		SpenderInputIndex: spend.Vin,
		SpendingHeight:    int32(spend.Status.BlockHeight),
	}
}

func (c *Core) bestBlock() (*chainhash.Hash, int32, *wire.BlockHeader) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := c.bestHash
	return &hash, c.bestHeight, c.bestHeader
}

type Notifier struct {
	core *Core
}

func NewNotifier(core *Core) *Notifier {
	return &Notifier{core: core}
}

func (n *Notifier) Start() error {
	return n.core.Start(context.Background())
}

func (n *Notifier) Started() bool {
	return n.core.Started()
}

func (n *Notifier) Stop() error {
	return n.core.Stop()
}

func (n *Notifier) RegisterConfirmationsNtfn(txid *chainhash.Hash,
	pkScript []byte, numConfs, _ uint32,
	opts ...chainntnfs.NotifierOption) (*chainntnfs.ConfirmationEvent,
	error) {

	ntfnOpts := chainntnfs.DefaultNotifierOptions()
	for _, opt := range opts {
		opt(ntfnOpts)
	}

	var id uint64
	event := chainntnfs.NewConfirmationEvent(numConfs, func() {
		n.core.mu.Lock()
		delete(n.core.confRegs, id)
		n.core.mu.Unlock()
	})

	reg := &confReg{
		txid:         txid,
		pkScript:     pkScript,
		numConfs:     numConfs,
		includeBlock: ntfnOpts.IncludeBlock,
		event:        event,
	}

	n.core.mu.Lock()
	id = n.core.nextRegID
	n.core.nextRegID++
	n.core.confRegs[id] = reg
	n.core.mu.Unlock()

	go n.core.recheckRegistrations()

	return event, nil
}

func (n *Notifier) RegisterSpendNtfn(outpoint *wire.OutPoint, _ []byte,
	_ uint32) (*chainntnfs.SpendEvent, error) {

	if outpoint == nil {
		return nil, fmt.Errorf("esplora spend registration requires an outpoint")
	}

	var id uint64
	event := chainntnfs.NewSpendEvent(func() {
		n.core.mu.Lock()
		delete(n.core.spendRegs, id)
		n.core.mu.Unlock()
	})

	reg := &spendReg{
		outpoint: outpoint,
		event:    event,
	}

	n.core.mu.Lock()
	id = n.core.nextRegID
	n.core.nextRegID++
	n.core.spendRegs[id] = reg
	n.core.mu.Unlock()

	go n.core.recheckRegistrations()

	return event, nil
}

func (n *Notifier) RegisterBlockEpochNtfn(best *chainntnfs.BlockEpoch) (
	*chainntnfs.BlockEpochEvent, error) {

	epochs := make(chan *chainntnfs.BlockEpoch, 20)

	var id uint64
	n.core.mu.Lock()
	id = n.core.nextRegID
	n.core.nextRegID++
	n.core.blockRegs[id] = &blockReg{epochs: epochs}
	n.core.mu.Unlock()

	cancel := func() {
		n.core.mu.Lock()
		delete(n.core.blockRegs, id)
		n.core.mu.Unlock()
		close(epochs)
	}

	go n.sendEpochBacklog(best, epochs)

	return &chainntnfs.BlockEpochEvent{
		Epochs: epochs,
		Cancel: cancel,
	}, nil
}

func (n *Notifier) sendEpochBacklog(best *chainntnfs.BlockEpoch,
	epochs chan *chainntnfs.BlockEpoch) {

	bestHash, bestHeight, bestHeader := n.core.bestBlock()
	start := bestHeight
	if best != nil {
		start = best.Height
	}

	for height := start; height <= bestHeight; height++ {
		hash := *bestHash
		header := bestHeader
		if height != bestHeight {
			var err error
			hash, err = n.core.client.BlockHashByHeight(
				context.Background(), height,
			)
			if err != nil {
				continue
			}
			header, err = n.core.client.RawBlockHeader(
				context.Background(), hash,
			)
			if err != nil {
				continue
			}
		}

		select {
		case epochs <- &chainntnfs.BlockEpoch{
			Hash:        &hash,
			Height:      height,
			BlockHeader: header,
		}:
		case <-n.core.quit:
			return
		}
	}
}

type View struct {
	core *Core
}

func NewView(core *Core) *View {
	return &View{core: core}
}

func (v *View) Start() error {
	return v.core.Start(context.Background())
}

func (v *View) Stop() error {
	return nil
}

func (v *View) FilteredBlocks() <-chan *chainview.FilteredBlock {
	return v.core.filtered
}

func (v *View) DisconnectedBlocks() <-chan *chainview.FilteredBlock {
	return v.core.disconnected
}

func (v *View) UpdateFilter(ops []graphdb.EdgePoint,
	updateHeight uint32) error {

	v.core.mu.Lock()
	for _, op := range ops {
		v.core.filterOps[op.OutPoint] = struct{}{}
	}
	bestHeight := v.core.bestHeight
	v.core.mu.Unlock()

	if updateHeight >= uint32(bestHeight) {
		return nil
	}

	for height := int32(updateHeight); height <= bestHeight; height++ {
		hash, err := v.core.client.BlockHashByHeight(
			context.Background(), height,
		)
		if err != nil {
			return err
		}
		block, err := v.core.filterBlock(context.Background(), hash)
		if err != nil {
			return err
		}
		if len(block.Transactions) == 0 {
			continue
		}

		select {
		case v.core.filtered <- block:
		case <-v.core.quit:
			return nil
		}
	}

	return nil
}

func (v *View) FilterBlock(blockHash *chainhash.Hash) (
	*chainview.FilteredBlock, error) {

	return v.core.filterBlock(context.Background(), *blockHash)
}

func (c *Core) filterBlock(ctx context.Context,
	hash chainhash.Hash) (*chainview.FilteredBlock, error) {

	meta, err := c.client.BlockMeta(ctx, hash)
	if err != nil {
		return nil, err
	}
	block, err := c.client.RawBlock(ctx, hash)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	filter := make(map[wire.OutPoint]struct{}, len(c.filterOps))
	for op := range c.filterOps {
		filter[op] = struct{}{}
	}
	c.mu.Unlock()

	var txs []*wire.MsgTx
	for _, tx := range block.Transactions {
		for _, txIn := range tx.TxIn {
			if _, ok := filter[txIn.PreviousOutPoint]; ok {
				txs = append(txs, tx)
				break
			}
		}
	}

	return &chainview.FilteredBlock{
		Hash:         hash,
		Height:       uint32(meta.Height),
		Transactions: txs,
	}, nil
}

type Source struct {
	core *Core
}

func NewSource(core *Core) *Source {
	return &Source{core: core}
}

func (s *Source) Start(ctx context.Context) error {
	return s.core.Start(ctx)
}

func (s *Source) Stop() {
	_ = s.core.Stop()
}

func (s *Source) WaitForShutdown() {
	s.core.wg.Wait()
}

func (s *Source) GetBestBlock() (*chainhash.Hash, int32, error) {
	hash, height, _ := s.core.bestBlock()
	return hash, height, nil
}

func (s *Source) GetBlock(hash *chainhash.Hash) (*wire.MsgBlock, error) {
	return s.core.client.RawBlock(context.Background(), *hash)
}

func (s *Source) GetBlockHash(height int64) (*chainhash.Hash, error) {
	hash, err := s.core.client.BlockHashByHeight(
		context.Background(), int32(height),
	)
	if err != nil {
		return nil, err
	}

	return &hash, nil
}

func (s *Source) GetBlockHeader(hash *chainhash.Hash) (
	*wire.BlockHeader, error) {

	return s.core.client.RawBlockHeader(context.Background(), *hash)
}

func (s *Source) GetUtxo(op *wire.OutPoint, pkScript []byte,
	_ uint32, cancel <-chan struct{}) (*wire.TxOut, error) {

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	go func() {
		select {
		case <-cancel:
			cancelCtx()

		case <-ctx.Done():
		}
	}()

	tx, err := s.core.client.RawTx(ctx, op.Hash)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", btcwallet.ErrOutputNotFound,
			err)
	}
	if uint32(len(tx.TxOut)) <= op.Index {
		return nil, btcwallet.ErrOutputNotFound
	}

	out := tx.TxOut[op.Index]
	if !bytes.Equal(out.PkScript, pkScript) {
		return nil, btcwallet.ErrOutputNotFound
	}

	spend, err := s.core.client.Outspend(ctx, *op)
	if err != nil {
		return nil, err
	}
	if spend.Spent {
		return nil, btcwallet.ErrOutputSpent
	}

	return out, nil
}

func (s *Source) IsCurrent() bool {
	return true
}

func (s *Source) FilterBlocks(req *walletchain.FilterBlocksRequest) (
	*walletchain.FilterBlocksResponse, error) {

	addrScripts := make(map[string]addressMatch)
	for scopedIdx, addr := range req.ExternalAddrs {
		pkScript, err := txscript.PayToAddrScript(addr)
		if err == nil {
			addrScripts[string(pkScript)] = addressMatch{
				scope:    scopedIdx.Scope,
				index:    scopedIdx.Index,
				external: true,
			}
		}
	}
	for scopedIdx, addr := range req.InternalAddrs {
		pkScript, err := txscript.PayToAddrScript(addr)
		if err == nil {
			addrScripts[string(pkScript)] = addressMatch{
				scope: scopedIdx.Scope,
				index: scopedIdx.Index,
			}
		}
	}

	for batchIdx, blockMeta := range req.Blocks {
		block, err := s.core.client.RawBlock(
			context.Background(), blockMeta.Hash,
		)
		if err != nil {
			return nil, err
		}

		resp := filterWalletBlock(
			block, blockMeta, uint32(batchIdx), addrScripts,
			req.WatchedOutPoints,
		)
		if resp != nil {
			return resp, nil
		}
	}

	return nil, nil
}

type addressMatch struct {
	scope    waddrmgr.KeyScope
	index    uint32
	external bool
}

func filterWalletBlock(block *wire.MsgBlock, meta wtxmgr.BlockMeta,
	batchIdx uint32, addrScripts map[string]addressMatch,
	watchedOPs map[wire.OutPoint]btcutil.Address) *walletchain.FilterBlocksResponse {

	foundExternal := make(map[waddrmgr.KeyScope]map[uint32]struct{})
	foundInternal := make(map[waddrmgr.KeyScope]map[uint32]struct{})
	foundOutpoints := make(map[wire.OutPoint]btcutil.Address)

	var relevant []*wire.MsgTx
	for _, tx := range block.Transactions {
		matched := false
		for _, out := range tx.TxOut {
			match, ok := addrScripts[string(out.PkScript)]
			if !ok {
				continue
			}
			matched = true
			if match.external {
				if foundExternal[match.scope] == nil {
					foundExternal[match.scope] = make(map[uint32]struct{})
				}
				foundExternal[match.scope][match.index] = struct{}{}
			} else {
				if foundInternal[match.scope] == nil {
					foundInternal[match.scope] = make(map[uint32]struct{})
				}
				foundInternal[match.scope][match.index] = struct{}{}
			}
		}
		for _, in := range tx.TxIn {
			addr, ok := watchedOPs[in.PreviousOutPoint]
			if !ok {
				continue
			}
			matched = true
			foundOutpoints[in.PreviousOutPoint] = addr
		}
		if matched {
			relevant = append(relevant, tx)
		}
	}

	if len(relevant) == 0 {
		return nil
	}

	return &walletchain.FilterBlocksResponse{
		BatchIndex:         batchIdx,
		BlockMeta:          meta,
		FoundExternalAddrs: foundExternal,
		FoundInternalAddrs: foundInternal,
		FoundOutPoints:     foundOutpoints,
		RelevantTxns:       relevant,
	}
}

func (s *Source) BlockStamp() (*waddrmgr.BlockStamp, error) {
	hash, height, header := s.core.bestBlock()
	return &waddrmgr.BlockStamp{
		Height:    height,
		Hash:      *hash,
		Timestamp: header.Timestamp,
	}, nil
}

func (s *Source) SendRawTransaction(tx *wire.MsgTx,
	_ bool) (*chainhash.Hash, error) {

	return s.core.client.BroadcastTx(context.Background(), tx)
}

func (s *Source) Rescan(startHash *chainhash.Hash, addrs []btcutil.Address,
	outpoints map[wire.OutPoint]btcutil.Address) error {

	ctx := context.Background()
	meta, err := s.core.client.BlockMeta(ctx, *startHash)
	if err != nil {
		return err
	}
	_, tipHeight, _ := s.core.bestBlock()
	tipHash, _, tipHeader := s.core.bestBlock()

	if len(addrs) == 0 && len(outpoints) == 0 {
		go s.core.sendWalletNotifications([]interface{}{
			&walletchain.RescanFinished{
				Hash:   tipHash,
				Height: tipHeight,
				Time:   tipHeader.Timestamp,
			},
		})

		return nil
	}

	addrScripts := make(map[string]struct{})
	for _, addr := range addrs {
		pkScript, err := txscript.PayToAddrScript(addr)
		if err == nil {
			addrScripts[string(pkScript)] = struct{}{}
		}
	}

	var pending []interface{}
	for height := meta.Height; height <= tipHeight; height++ {
		hash, err := s.core.client.BlockHashByHeight(ctx, height)
		if err != nil {
			return err
		}
		block, err := s.core.client.RawBlock(ctx, hash)
		if err != nil {
			return err
		}
		header, err := s.core.client.RawBlockHeader(ctx, hash)
		if err != nil {
			return err
		}

		blockMeta := wtxmgr.BlockMeta{
			Block: wtxmgr.Block{
				Hash:   hash,
				Height: height,
			},
			Time: header.Timestamp,
		}

		var relevant []*wtxmgr.TxRecord
		for _, tx := range block.Transactions {
			if !txMatchesRescan(tx, addrScripts, outpoints) {
				continue
			}
			rec, err := wtxmgr.NewTxRecordFromMsgTx(tx, header.Timestamp)
			if err == nil {
				relevant = append(relevant, rec)
			}
		}

		pending = append(pending, walletchain.FilteredBlockConnected{
			Block:       &blockMeta,
			RelevantTxs: relevant,
		})
		pending = append(pending, walletchain.BlockConnected(blockMeta))
	}

	pending = append(pending, &walletchain.RescanFinished{
		Hash:   tipHash,
		Height: tipHeight,
		Time:   tipHeader.Timestamp,
	})
	go s.core.sendWalletNotifications(pending)

	return nil
}

func (c *Core) sendWalletNotifications(notifications []interface{}) {
	for _, notification := range notifications {
		select {
		case c.notifications <- notification:
		case <-c.quit:
			return
		}
	}
}

func txMatchesRescan(tx *wire.MsgTx, scripts map[string]struct{},
	outpoints map[wire.OutPoint]btcutil.Address) bool {

	for _, out := range tx.TxOut {
		if _, ok := scripts[string(out.PkScript)]; ok {
			return true
		}
	}
	for _, in := range tx.TxIn {
		if _, ok := outpoints[in.PreviousOutPoint]; ok {
			return true
		}
	}

	return false
}

func (s *Source) NotifyReceived(addrs []btcutil.Address) error {
	s.core.mu.Lock()
	for _, addr := range addrs {
		s.core.watchedAddr[addr.String()] = addr
	}
	s.core.mu.Unlock()

	return nil
}

func (s *Source) NotifyBlocks() error {
	return nil
}

func (s *Source) Notifications() <-chan interface{} {
	return s.core.notifications
}

func (s *Source) BackEnd() string {
	return backendName
}

func (s *Source) TestMempoolAccept(txns []*wire.MsgTx,
	maxFeeRate float64) ([]*btcjson.TestMempoolAcceptResult, error) {

	return s.core.client.TestMempoolAccept(
		context.Background(), txns, maxFeeRate,
	)
}

func (s *Source) MapRPCErr(err error) error {
	return err
}

type Estimator struct {
	client *Client
}

func NewEstimator(client *Client) *Estimator {
	return &Estimator{client: client}
}

func (e *Estimator) Start() error {
	return nil
}

func (e *Estimator) Stop() error {
	return nil
}

func (e *Estimator) RelayFeePerKW() chainfee.SatPerKWeight {
	return chainfee.FeePerKwFloor
}

func (e *Estimator) EstimateFeePerKW(numBlocks uint32) (
	chainfee.SatPerKWeight, error) {

	estimates, err := e.client.FeeEstimates(context.Background())
	if err != nil {
		return 0, err
	}

	var (
		bestRate   float64
		bestTarget uint32 = math.MaxUint32
	)
	for targetStr, rate := range estimates {
		target, err := strconv.ParseUint(targetStr, 10, 32)
		if err != nil {
			continue
		}
		t := uint32(target)
		if t >= numBlocks && t < bestTarget {
			bestTarget = t
			bestRate = rate
		}
	}
	if bestTarget == math.MaxUint32 {
		for _, rate := range estimates {
			if bestRate == 0 || rate < bestRate {
				bestRate = rate
			}
		}
	}
	if bestRate < 1 {
		bestRate = 1
	}

	return chainfee.SatPerKVByte(math.Ceil(bestRate) * 1000).FeePerKWeight(),
		nil
}

var _ chainntnfs.ChainNotifier = (*Notifier)(nil)
var _ chainview.FilteredChainView = (*View)(nil)
var _ walletchain.Interface = (*Source)(nil)
var _ chainfee.Estimator = (*Estimator)(nil)
