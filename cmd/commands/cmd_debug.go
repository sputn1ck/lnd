package commands

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/andybalholm/brotli"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/lightningnetwork/lnd"
	"github.com/lightningnetwork/lnd/lnencrypt"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/urfave/cli"
	"google.golang.org/protobuf/proto"
)

var getDebugInfoCommand = cli.Command{
	Name:     "getdebuginfo",
	Category: "Debug",
	Usage:    "Returns debug information related to the active daemon.",
	Action:   actionDecorator(getDebugInfo),
}

func getDebugInfo(ctx *cli.Context) error {
	ctxc := getContext()
	client, cleanUp := getClient(ctx)
	defer cleanUp()

	req := &lnrpc.GetDebugInfoRequest{}
	resp, err := client.GetDebugInfo(ctxc, req)
	if err != nil {
		return err
	}

	printRespJSON(resp)

	return nil
}

type DebugPackage struct {
	EphemeralPubKey  string `json:"ephemeral_public_key"`
	EncryptedPayload string `json:"encrypted_payload"`
}

var encryptDebugPackageCommand = cli.Command{
	Name:     "encryptdebugpackage",
	Category: "Debug",
	Usage:    "Collects a package of debug information and encrypts it.",
	Description: `
	When requesting support with lnd, it's often required to submit a lot of
	debug information to the developer in order to track down a problem.
	This command will collect all the relevant information and encrypt it
	using the provided public key. The resulting file can then be sent to
	the developer for further analysis.
	Because the file is encrypted, it is safe to send it over insecure
	channels or upload it to a GitHub issue.

	The file by default contains the output of the following commands:
	- lncli getinfo
	- lncli getdebuginfo
	- lncli getnetworkinfo

	By specifying the following flags, additional information can be added
	to the file (usually this will be requested by the developer depending
	on the issue at hand):
		--peers:
			- lncli listpeers
		--onchain:
			- lncli listunspent
			- lncli listchaintxns
		--channels:
			- lncli listchannels
			- lncli pendingchannels
			- lncli closedchannels

	Use 'lncli encryptdebugpackage 0xxxxxx... > package.txt' to write the
	encrypted package to a file called package.txt.
	`,
	ArgsUsage: "pubkey [--output_file F]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name: "pubkey",
			Usage: "the public key to encrypt the information " +
				"for (hex-encoded, e.g. 02aabb..), this " +
				"should be provided to you by the issue " +
				"tracker or developer you're requesting " +
				"support from",
		},
		cli.StringFlag{
			Name: "output_file",
			Usage: "(optional) the file to write the encrypted " +
				"package to; if not specified, the debug " +
				"package is printed to stdout",
		},
		cli.BoolFlag{
			Name: "peers",
			Usage: "include information about connected peers " +
				"(lncli listpeers)",
		},
		cli.BoolFlag{
			Name: "onchain",
			Usage: "include information about on-chain " +
				"transactions (lncli listunspent, " +
				"lncli listchaintxns)",
		},
		cli.BoolFlag{
			Name: "channels",
			Usage: "include information about channels " +
				"(lncli listchannels, lncli pendingchannels, " +
				"lncli closedchannels)",
		},
	},
	Action: actionDecorator(encryptDebugPackage),
}

func encryptDebugPackage(ctx *cli.Context) error {
	if ctx.NArg() == 0 && ctx.NumFlags() == 0 {
		return cli.ShowCommandHelp(ctx, "encryptdebugpackage")
	}

	var (
		args        = ctx.Args()
		pubKeyBytes []byte
		err         error
	)
	switch {
	case ctx.IsSet("pubkey"):
		pubKeyBytes, err = hex.DecodeString(ctx.String("pubkey"))
	case args.Present():
		pubKeyBytes, err = hex.DecodeString(args.First())
	}
	if err != nil {
		return fmt.Errorf("unable to decode pubkey argument: %w", err)
	}

	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("unable to parse pubkey: %w", err)
	}

	// Collect the information we want to send from the daemon.
	payload, err := collectDebugPackageInfo(ctx)
	if err != nil {
		return fmt.Errorf("unable to collect debug package "+
			"information: %w", err)
	}

	// We've collected the information we want to send, but before
	// encrypting it, we want to compress it as much as possible to reduce
	// the size of the final payload.
	var (
		compressBuf bytes.Buffer
		options     = brotli.WriterOptions{
			Quality: brotli.BestCompression,
		}
		writer = brotli.NewWriterOptions(&compressBuf, options)
	)
	_, err = writer.Write(payload)
	if err != nil {
		return fmt.Errorf("unable to compress payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("unable to compress payload: %w", err)
	}

	// Now we have the full payload that we want to encrypt, so we'll create
	// an ephemeral keypair to encrypt the payload with.
	localKey, err := btcec.NewPrivateKey()
	if err != nil {
		return fmt.Errorf("unable to generate local key: %w", err)
	}

	enc, err := lnencrypt.ECDHEncrypter(localKey, pubKey)
	if err != nil {
		return fmt.Errorf("unable to create encrypter: %w", err)
	}

	var cipherBuf bytes.Buffer
	err = enc.EncryptPayloadToWriter(compressBuf.Bytes(), &cipherBuf)
	if err != nil {
		return fmt.Errorf("unable to encrypt payload: %w", err)
	}

	response := DebugPackage{
		EphemeralPubKey: hex.EncodeToString(
			localKey.PubKey().SerializeCompressed(),
		),
		EncryptedPayload: hex.EncodeToString(
			cipherBuf.Bytes(),
		),
	}

	// If the user specified an output file, we'll write the encrypted
	// payload to that file.
	if ctx.IsSet("output_file") {
		fileName := lnd.CleanAndExpandPath(ctx.String("output_file"))
		jsonBytes, err := json.Marshal(response)
		if err != nil {
			return fmt.Errorf("unable to encode JSON: %w", err)
		}

		return os.WriteFile(fileName, jsonBytes, 0644)
	}

	// Finally, we'll print out the final payload as a JSON if no output
	// file was specified.
	printJSON(response)

	return nil
}

// maxLogSize is the maximum size of log data to include in the debug package.
// If the log file is larger than this, we'll only include the last portion.
const maxLogSize = 10 * 1024 * 1024 // 10 MB

// collectDebugPackageInfo collects the information we want to send to the
// developer(s) from the daemon.
func collectDebugPackageInfo(ctx *cli.Context) ([]byte, error) {
	ctxc := getContext()
	client, cleanUp := getClient(ctx)
	defer cleanUp()

	info, err := client.GetInfo(ctxc, &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("error getting info: %w", err)
	}

	debugInfo, err := client.GetDebugInfo(
		ctxc, &lnrpc.GetDebugInfoRequest{},
	)
	if err != nil {
		return nil, fmt.Errorf("error getting debug info: %w", err)
	}

	// If the log is too large, truncate it to include only the most recent
	// entries. This helps prevent memory issues and reduces upload size.
	if len(debugInfo.Log) > 0 {
		totalSize := 0
		for _, line := range debugInfo.Log {
			totalSize += len(line)
		}
		
		if totalSize > maxLogSize {
			// Calculate how many lines to keep from the end
			keptSize := 0
			startIdx := len(debugInfo.Log) - 1
			for i := len(debugInfo.Log) - 1; i >= 0; i-- {
				lineSize := len(debugInfo.Log[i])
				if keptSize+lineSize > maxLogSize {
					startIdx = i + 1
					break
				}
				keptSize += lineSize
			}
			
			// Keep only the most recent log entries
			truncatedLog := make([]string, 0, len(debugInfo.Log)-startIdx+1)
			truncatedLog = append(truncatedLog, 
				fmt.Sprintf("... log truncated, showing last %d KB ...", 
					keptSize/1024))
			truncatedLog = append(truncatedLog, debugInfo.Log[startIdx:]...)
			debugInfo.Log = truncatedLog
		}
	}

	networkInfo, err := client.GetNetworkInfo(
		ctxc, &lnrpc.NetworkInfoRequest{},
	)
	if err != nil {
		return nil, fmt.Errorf("error getting network info: %w", err)
	}

	var payloadBuf bytes.Buffer
	addToBuf := func(msgs ...proto.Message) error {
		for _, msg := range msgs {
			jsonBytes, err := lnrpc.ProtoJSONMarshalOpts.Marshal(
				msg,
			)
			if err != nil {
				return fmt.Errorf("error encoding response: %w",
					err)
			}

			payloadBuf.Write(jsonBytes)
		}

		return nil
	}

	if err := addToBuf(info); err != nil {
		return nil, err
	}
	if err := addToBuf(debugInfo); err != nil {
		return nil, err
	}
	if err := addToBuf(info, debugInfo, networkInfo); err != nil {
		return nil, err
	}

	// Add optional information to the payload.
	if ctx.Bool("peers") {
		peers, err := client.ListPeers(ctxc, &lnrpc.ListPeersRequest{
			LatestError: true,
		})
		if err != nil {
			return nil, fmt.Errorf("error getting peers: %w", err)
		}
		if err := addToBuf(peers); err != nil {
			return nil, err
		}
	}

	if ctx.Bool("onchain") {
		unspent, err := client.ListUnspent(
			ctxc, &lnrpc.ListUnspentRequest{
				MaxConfs: math.MaxInt32,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error getting unspent: %w", err)
		}
		chainTxns, err := client.GetTransactions(
			ctxc, &lnrpc.GetTransactionsRequest{},
		)
		if err != nil {
			return nil, fmt.Errorf("error getting chain txns: %w",
				err)
		}
		if err := addToBuf(unspent, chainTxns); err != nil {
			return nil, err
		}
	}

	if ctx.Bool("channels") {
		channels, err := client.ListChannels(
			ctxc, &lnrpc.ListChannelsRequest{},
		)
		if err != nil {
			return nil, fmt.Errorf("error getting channels: %w",
				err)
		}
		pendingChannels, err := client.PendingChannels(
			ctxc, &lnrpc.PendingChannelsRequest{},
		)
		if err != nil {
			return nil, fmt.Errorf("error getting pending "+
				"channels: %w", err)
		}
		closedChannels, err := client.ClosedChannels(
			ctxc, &lnrpc.ClosedChannelsRequest{},
		)
		if err != nil {
			return nil, fmt.Errorf("error getting closed "+
				"channels: %w", err)
		}
		if err := addToBuf(
			channels, pendingChannels, closedChannels,
		); err != nil {
			return nil, err
		}
	}

	return payloadBuf.Bytes(), nil
}

var decryptDebugPackageCommand = cli.Command{
	Name:     "decryptdebugpackage",
	Category: "Debug",
	Usage:    "Decrypts a package of debug information.",
	Description: `
	Decrypt a debug package that was created with the encryptdebugpackage
	command. Decryption requires the private key that corresponds to the
	public key the package was encrypted to.
	The command expects the encrypted package JSON to be provided on stdin.
	If decryption is successful, the information will be printed to stdout.

	Use 'lncli decryptdebugpackage 0xxxxxx... < package.txt > decrypted.txt'
	to read the encrypted package from a file called package.txt and to
	write the decrypted content to a file called decrypted.txt.
	`,
	ArgsUsage: "privkey [--input_file F]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name: "privkey",
			Usage: "the hex encoded private key to decrypt the " +
				"debug package",
		},
		cli.StringFlag{
			Name: "input_file",
			Usage: "(optional) the file to read the encrypted " +
				"package from; if not specified, the debug " +
				"package is read from stdin",
		},
	},
	Action: actionDecorator(decryptDebugPackage),
}

func decryptDebugPackage(ctx *cli.Context) error {
	if ctx.NArg() == 0 && ctx.NumFlags() == 0 {
		return cli.ShowCommandHelp(ctx, "decryptdebugpackage")
	}

	var (
		args         = ctx.Args()
		privKeyBytes []byte
		err          error
	)
	switch {
	case ctx.IsSet("pubkey"):
		privKeyBytes, err = hex.DecodeString(ctx.String("pubkey"))
	case args.Present():
		privKeyBytes, err = hex.DecodeString(args.First())
	}
	if err != nil {
		return fmt.Errorf("unable to decode privkey argument: %w", err)
	}

	privKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)

	// Read the file from stdin and decode the JSON into a DebugPackage.
	var pkg DebugPackage
	if ctx.IsSet("input_file") {
		fileName := lnd.CleanAndExpandPath(ctx.String("input_file"))
		jsonBytes, err := os.ReadFile(fileName)
		if err != nil {
			return fmt.Errorf("unable to read file '%s': %w",
				fileName, err)
		}

		err = json.Unmarshal(jsonBytes, &pkg)
		if err != nil {
			return fmt.Errorf("unable to decode JSON: %w", err)
		}
	} else {
		err = json.NewDecoder(os.Stdin).Decode(&pkg)
		if err != nil {
			return fmt.Errorf("unable to decode JSON: %w", err)
		}
	}

	// Decode the ephemeral public key and encrypted payload.
	ephemeralPubKeyBytes, err := hex.DecodeString(pkg.EphemeralPubKey)
	if err != nil {
		return fmt.Errorf("unable to decode ephemeral public key: %w",
			err)
	}
	encryptedPayloadBytes, err := hex.DecodeString(pkg.EncryptedPayload)
	if err != nil {
		return fmt.Errorf("unable to decode encrypted payload: %w", err)
	}

	// Parse the ephemeral public key and create an encrypter.
	ephemeralPubKey, err := btcec.ParsePubKey(ephemeralPubKeyBytes)
	if err != nil {
		return fmt.Errorf("unable to parse ephemeral public key: %w",
			err)
	}
	enc, err := lnencrypt.ECDHEncrypter(privKey, ephemeralPubKey)
	if err != nil {
		return fmt.Errorf("unable to create encrypter: %w", err)
	}

	// Decrypt the payload.
	decryptedPayload, err := enc.DecryptPayloadFromReader(
		bytes.NewReader(encryptedPayloadBytes),
	)
	if err != nil {
		return fmt.Errorf("unable to decrypt payload: %w", err)
	}

	// Decompress the payload.
	reader := brotli.NewReader(bytes.NewBuffer(decryptedPayload))
	decompressedPayload, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("unable to decompress payload: %w", err)
	}

	fmt.Println(string(decompressedPayload))

	return nil
}

// GitHub API endpoint for creating gists
const githubGistAPI = "https://api.github.com/gists"

// EncryptedDebugPackage represents the encrypted debug package with keys for each developer
type EncryptedDebugPackage struct {
	EncryptedData string            `json:"encrypted_data"` // Base64 encoded AES encrypted data
	Nonce         string            `json:"nonce"`          // Base64 encoded nonce for AES-GCM
	EncryptedKeys map[string]string `json:"encrypted_keys"` // GPG encrypted AES keys per developer
	Timestamp     int64             `json:"timestamp"`
	Version       string            `json:"version"`
}

// GistRequest represents a GitHub Gist creation request
type GistRequest struct {
	Description string              `json:"description"`
	Public      bool                `json:"public"`
	Files       map[string]GistFile `json:"files"`
}

// GistFile represents a file in a GitHub Gist
type GistFile struct {
	Content string `json:"content"`
}

// GistResponse represents the response from GitHub Gist API
type GistResponse struct {
	ID      string `json:"id"`
	HTMLURL string `json:"html_url"`
}

var submitDebugPackageCommand = cli.Command{
	Name:     "submitdebugpackage",
	Category: "Debug",
	Usage:    "Collects, encrypts and uploads debug information for Lightning Labs support.",
	Description: `
	This command automates the process of collecting debug information,
	encrypting it, and uploading it to a paste service for Lightning Labs support.
	
	The command will:
	1. Collect the same information as 'encryptdebugpackage'
	2. Compress the data using brotli
	3. Encrypt it using AES-256-GCM with a random key
	4. Encrypt the AES key with each Lightning Labs developer's GPG public key
	5. Upload to a paste service (GitHub Gist, dpaste, or file output)
	6. Return a URL that can be referenced in GitHub issues
	
	Upload priority:
	- GitHub Gist (if token provided via --github-token or GITHUB_TOKEN env)
	- dpaste.com (anonymous, no token required, 7-day expiry)
	- Local file output (if --output-file specified or no service available)
	
	The URL returned can be shared when opening support issues. Any
	Lightning Labs developer with their GPG private key can decrypt the package.
	`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name: "github-token",
			Usage: "GitHub personal access token for authenticated uploads " +
				"(can also use GITHUB_TOKEN env var)",
		},
		cli.StringFlag{
			Name: "service",
			Usage: "Paste service to use: github, dpaste, or file " +
				"(auto-detects based on available tokens if not specified)",
		},
		cli.StringFlag{
			Name: "output-file",
			Usage: "Write to a local file instead of uploading " +
				"(used automatically if no paste service is available)",
		},
		cli.StringFlag{
			Name: "gpg-key",
			Usage: "Path to a specific GPG public key file to use for encryption " +
				"(for testing; normally uses all keys in scripts/keys)",
		},
		cli.BoolFlag{
			Name: "peers",
			Usage: "include information about connected peers " +
				"(lncli listpeers)",
		},
		cli.BoolFlag{
			Name: "onchain",
			Usage: "include information about on-chain " +
				"transactions (lncli listunspent, " +
				"lncli listchaintxns)",
		},
		cli.BoolFlag{
			Name: "channels",
			Usage: "include information about channels " +
				"(lncli listchannels, lncli pendingchannels, " +
				"lncli closedchannels)",
		},
		cli.DurationFlag{
			Name:  "timeout",
			Usage: "timeout for the HTTP request",
			Value: 5 * time.Minute,
		},
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "collect and encrypt the package but don't submit it",
		},
	},
	Action: actionDecorator(submitDebugPackage),
}

func submitDebugPackage(ctx *cli.Context) error {
	// Collect debug information
	fmt.Println("Collecting debug information...")
	payload, err := collectDebugPackageInfo(ctx)
	if err != nil {
		return fmt.Errorf("unable to collect debug package "+
			"information: %w", err)
	}

	// Compress the payload
	fmt.Println("Compressing debug package...")
	var (
		compressBuf bytes.Buffer
		options     = brotli.WriterOptions{
			Quality: brotli.BestCompression,
		}
		writer = brotli.NewWriterOptions(&compressBuf, options)
	)
	_, err = writer.Write(payload)
	if err != nil {
		return fmt.Errorf("unable to compress payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("unable to compress payload: %w", err)
	}

	// Generate AES-256 key for symmetric encryption
	fmt.Println("Generating encryption key...")
	aesKey := make([]byte, 32) // 256 bits
	if _, err := rand.Read(aesKey); err != nil {
		return fmt.Errorf("unable to generate AES key: %w", err)
	}

	// Encrypt the compressed payload with AES-256-GCM
	fmt.Println("Encrypting debug package...")
	encryptedData, nonce, err := encryptWithAES(compressBuf.Bytes(), aesKey)
	if err != nil {
		return fmt.Errorf("unable to encrypt payload: %w", err)
	}

	// Load GPG public keys
	fmt.Println("Loading GPG keys...")
	var encryptedKeys map[string]string
	if gpgKeyPath := ctx.String("gpg-key"); gpgKeyPath != "" {
		// Use specific GPG key for testing
		encryptedKeys, err = encryptKeyWithSpecificGPG(aesKey, gpgKeyPath)
		if err != nil {
			return fmt.Errorf("unable to encrypt AES key with specific GPG key: %w", err)
		}
	} else {
		// Use all Lightning Labs GPG keys
		encryptedKeys, err = encryptKeyWithGPG(aesKey)
		if err != nil {
			return fmt.Errorf("unable to encrypt AES key with GPG: %w", err)
		}
	}

	// Get version info
	ctxc := getContext()
	client, cleanUp := getClient(ctx)
	defer cleanUp()
	
	info, err := client.GetInfo(ctxc, &lnrpc.GetInfoRequest{})
	if err != nil {
		// Non-fatal, continue without version info
		info = &lnrpc.GetInfoResponse{Version: "unknown"}
	}

	// Create the encrypted package
	encPackage := EncryptedDebugPackage{
		EncryptedData: base64.StdEncoding.EncodeToString(encryptedData),
		Nonce:         base64.StdEncoding.EncodeToString(nonce),
		EncryptedKeys: encryptedKeys,
		Timestamp:     time.Now().Unix(),
		Version:       info.Version,
	}

	// If dry-run, just show the package size and exit
	if ctx.Bool("dry-run") {
		packageJSON, err := json.Marshal(encPackage)
		if err != nil {
			return fmt.Errorf("unable to marshal package: %w", err)
		}
		fmt.Printf("Dry run complete. Package size: %d bytes\n", 
			len(packageJSON))
		fmt.Printf("Encrypted for %d Lightning Labs developers\n", 
			len(encryptedKeys))
		fmt.Println("Package ready for upload (not uploaded due to --dry-run)")
		return nil
	}

	// Upload the package
	uploadURL, err := uploadPackage(ctx, encPackage)
	if err != nil {
		return fmt.Errorf("unable to upload package: %w", err)
	}

	// Display the result for the user to reference
	fmt.Println("\n========================================")
	fmt.Printf("Debug package uploaded successfully!\n")
	fmt.Printf("Location: %s\n", uploadURL)
	fmt.Println("Please include this URL/path when opening a GitHub issue.")
	fmt.Println("Any Lightning Labs developer can decrypt this package.")
	fmt.Println("========================================")

	return nil
}

// encryptWithAES encrypts data using AES-256-GCM
func encryptWithAES(data []byte, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)
	return ciphertext, nonce, nil
}

// encryptKeyWithSpecificGPG encrypts the AES key with a specific GPG public key
func encryptKeyWithSpecificGPG(aesKey []byte, keyPath string) (map[string]string, error) {
	encryptedKeys := make(map[string]string)
	
	keyFile, err := os.Open(keyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open GPG key file: %w", err)
	}
	defer keyFile.Close()

	// Read the armored public key
	block, err := armor.Decode(keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to decode armored key: %w", err)
	}

	// Parse the public key
	reader := packet.NewReader(block.Body)
	entity, err := openpgp.ReadEntity(reader)
	if err != nil {
		return nil, fmt.Errorf("unable to parse public key: %w", err)
	}

	// Encrypt the AES key with this public key
	var encBuf bytes.Buffer
	armorWriter, err := armor.Encode(&encBuf, "PGP MESSAGE", nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create armor writer: %w", err)
	}

	plainWriter, err := openpgp.Encrypt(armorWriter, 
		[]*openpgp.Entity{entity}, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to encrypt: %w", err)
	}

	if _, err := plainWriter.Write(aesKey); err != nil {
		return nil, fmt.Errorf("unable to write encrypted key: %w", err)
	}
	
	plainWriter.Close()
	armorWriter.Close()

	// Store with filename as key
	keyName := filepath.Base(keyPath)
	keyName = strings.TrimSuffix(keyName, filepath.Ext(keyName))
	encryptedKeys[keyName] = encBuf.String()
	
	fmt.Printf("  Encrypted for: %s\n", keyName)
	
	return encryptedKeys, nil
}

// encryptKeyWithGPG encrypts the AES key with all Lightning Labs GPG public keys
func encryptKeyWithGPG(aesKey []byte) (map[string]string, error) {
	encryptedKeys := make(map[string]string)
	
	// Find the scripts/keys directory relative to the current working directory
	keysDir := filepath.Join("scripts", "keys")
	
	// Check if we're in a subdirectory and need to go up
	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		// Try from parent directories (in case we're in cmd/commands)
		keysDir = filepath.Join("..", "..", "scripts", "keys")
		if _, err := os.Stat(keysDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("unable to find scripts/keys directory")
		}
	}
	
	// Read all .asc files from the keys directory
	entries, err := os.ReadDir(keysDir)
	if err != nil {
		return nil, fmt.Errorf("unable to read keys directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".asc") {
			continue
		}
		
		// Skip README.md.asc or other non-key files
		if entry.Name() == "README.md" {
			continue
		}

		keyPath := filepath.Join(keysDir, entry.Name())
		keyFile, err := os.Open(keyPath)
		if err != nil {
			return nil, fmt.Errorf("unable to open key file %s: %w", 
				entry.Name(), err)
		}
		defer keyFile.Close()

		// Read the armored public key
		block, err := armor.Decode(keyFile)
		if err != nil {
			return nil, fmt.Errorf("unable to decode armored key %s: %w",
				entry.Name(), err)
		}

		// Parse the public key
		reader := packet.NewReader(block.Body)
		entity, err := openpgp.ReadEntity(reader)
		if err != nil {
			return nil, fmt.Errorf("unable to parse public key %s: %w",
				entry.Name(), err)
		}

		// Encrypt the AES key with this public key
		var encBuf bytes.Buffer
		armorWriter, err := armor.Encode(&encBuf, "PGP MESSAGE", nil)
		if err != nil {
			return nil, fmt.Errorf("unable to create armor writer: %w", err)
		}

		plainWriter, err := openpgp.Encrypt(armorWriter, 
			[]*openpgp.Entity{entity}, nil, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to encrypt for %s: %w",
				entry.Name(), err)
		}

		if _, err := plainWriter.Write(aesKey); err != nil {
			return nil, fmt.Errorf("unable to write encrypted key: %w", err)
		}
		
		plainWriter.Close()
		armorWriter.Close()

		// Store the encrypted key
		keyName := strings.TrimSuffix(entry.Name(), ".asc")
		encryptedKeys[keyName] = encBuf.String()
		
		fmt.Printf("  Encrypted for: %s\n", keyName)
	}

	if len(encryptedKeys) == 0 {
		return nil, fmt.Errorf("no GPG keys found in %s", keysDir)
	}

	return encryptedKeys, nil
}

// uploadPackage uploads the package to an appropriate service
func uploadPackage(ctx *cli.Context, pkg EncryptedDebugPackage) (string, error) {
	// Marshal the package to JSON
	packageJSON, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("unable to marshal package: %w", err)
	}

	// Determine which service to use
	service := ctx.String("service")
	githubToken := ctx.String("github-token")
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}

	// Auto-detect service if not specified
	if service == "" {
		if githubToken != "" {
			service = "github"
		} else if ctx.String("output-file") != "" {
			service = "file"
		} else {
			service = "dpaste"
		}
	}

	switch service {
	case "github":
		if githubToken == "" {
			return "", fmt.Errorf("GitHub token required for GitHub Gist upload")
		}
		fmt.Println("Uploading to GitHub Gist...")
		return uploadToGist(packageJSON, githubToken, pkg.Timestamp)

	case "dpaste":
		fmt.Println("Uploading to dpaste.com...")
		return uploadToDpaste(packageJSON)

	case "file":
		outputFile := ctx.String("output-file")
		if outputFile == "" {
			outputFile = fmt.Sprintf("debug-package-%d.json", pkg.Timestamp)
		}
		fmt.Printf("Writing to file: %s\n", outputFile)
		return writeToFile(packageJSON, outputFile)

	default:
		return "", fmt.Errorf("unknown service: %s", service)
	}
}

// uploadToGist uploads the encrypted package to GitHub Gist
func uploadToGist(packageJSON []byte, githubToken string, timestamp int64) (string, error) {
	// Create gist request
	gistReq := GistRequest{
		Description: fmt.Sprintf("LND Debug Package - %s", 
			time.Unix(timestamp, 0).Format(time.RFC3339)),
		Public: false, // Private gist
		Files: map[string]GistFile{
			"debug-package.json": {
				Content: string(packageJSON),
			},
		},
	}

	reqJSON, err := json.Marshal(gistReq)
	if err != nil {
		return "", fmt.Errorf("unable to marshal gist request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", githubGistAPI, 
		bytes.NewReader(reqJSON))
	if err != nil {
		return "", fmt.Errorf("unable to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+githubToken)

	// Submit the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to submit gist: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error (status %d): %s",
			resp.StatusCode, string(body))
	}

	// Parse response
	var gistResp GistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return "", fmt.Errorf("unable to decode response: %w", err)
	}

	return gistResp.HTMLURL, nil
}

// uploadToDpaste uploads to dpaste.com (no API key required)
func uploadToDpaste(packageJSON []byte) (string, error) {
	// Create form data
	formData := url.Values{}
	formData.Set("content", string(packageJSON))
	formData.Set("syntax", "json")
	formData.Set("expiry_days", "7")
	
	resp, err := http.PostForm("https://dpaste.com/api/v2/", formData)
	if err != nil {
		return "", fmt.Errorf("unable to upload to dpaste: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("dpaste error (status %d): %s",
			resp.StatusCode, string(body))
	}

	// dpaste returns the URL in the Location header
	location := resp.Header.Get("Location")
	if location == "" {
		// Try to read from body
		body, _ := io.ReadAll(resp.Body)
		location = strings.TrimSpace(string(body))
	}

	if location == "" {
		return "", fmt.Errorf("dpaste did not return a URL")
	}

	// Add .json extension for raw access
	return location + ".json", nil
}

// writeToFile writes the package to a local file
func writeToFile(packageJSON []byte, filename string) (string, error) {
	err := os.WriteFile(filename, packageJSON, 0644)
	if err != nil {
		return "", fmt.Errorf("unable to write file: %w", err)
	}
	
	// Return absolute path
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return filename, nil
	}
	
	return absPath, nil
}
