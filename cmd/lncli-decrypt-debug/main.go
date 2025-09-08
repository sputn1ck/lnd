package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/andybalholm/brotli"
)

// EncryptedDebugPackage represents the encrypted debug package
type EncryptedDebugPackage struct {
	EncryptedData string            `json:"encrypted_data"`
	Nonce         string            `json:"nonce"`
	EncryptedKeys map[string]string `json:"encrypted_keys"`
	Timestamp     int64             `json:"timestamp"`
	Version       string            `json:"version"`
}

func main() {
	var (
		input  = flag.String("input", "", "Gist URL, dpaste URL, or path to JSON file")
		output = flag.String("output", "debug-package-decrypted.json", "Output file path")
		keyID  = flag.String("key", "", "GPG key ID to use (optional)")
	)
	flag.Parse()

	if *input == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -input <url-or-file> [-output <file>] [-key <id>]\n", os.Args[0])
		os.Exit(1)
	}

	// Load the encrypted package
	pkg, err := loadPackage(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading package: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Package version: %s\n", pkg.Version)
	fmt.Printf("Available keys: %v\n", getKeys(pkg.EncryptedKeys))

	// Decrypt the AES key using GPG
	aesKey, keyUsed, err := decryptAESKey(pkg.EncryptedKeys, *keyID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decrypting AES key: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Decrypted with key: %s\n", keyUsed)

	// Decode the encrypted data and nonce
	encryptedData, err := base64.StdEncoding.DecodeString(pkg.EncryptedData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding encrypted data: %v\n", err)
		os.Exit(1)
	}

	nonce, err := base64.StdEncoding.DecodeString(pkg.Nonce)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding nonce: %v\n", err)
		os.Exit(1)
	}

	// Decrypt with AES-256-GCM
	plaintext, err := decryptAESGCM(encryptedData, aesKey, nonce)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decrypting data: %v\n", err)
		os.Exit(1)
	}

	// Decompress the data
	reader := brotli.NewReader(bytes.NewReader(plaintext))
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decompressing data: %v\n", err)
		os.Exit(1)
	}

	// Write to output file
	err = os.WriteFile(*output, decompressed, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully decrypted to: %s\n", *output)
}

func loadPackage(input string) (*EncryptedDebugPackage, error) {
	var data []byte
	var err error

	if strings.HasPrefix(input, "https://gist.github.com/") {
		// Extract gist ID and fetch
		parts := strings.Split(input, "/")
		gistID := parts[len(parts)-1]
		
		// Try raw URL first
		rawURL := fmt.Sprintf("https://gist.githubusercontent.com/raw/%s", gistID)
		resp, err := http.Get(rawURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else if strings.Contains(input, "dpaste.com") {
		// Handle dpaste URLs
		if !strings.HasSuffix(input, ".json") {
			input = input + ".json"
		}
		resp, err := http.Get(input)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		// Read from file
		data, err = os.ReadFile(input)
		if err != nil {
			return nil, err
		}
	}

	var pkg EncryptedDebugPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	return &pkg, nil
}

func getKeys(encryptedKeys map[string]string) []string {
	keys := make([]string, 0, len(encryptedKeys))
	for k := range encryptedKeys {
		keys = append(keys, k)
	}
	return keys
}

func decryptAESKey(encryptedKeys map[string]string, preferredKey string) ([]byte, string, error) {
	// Try preferred key first if specified
	if preferredKey != "" {
		if encKey, ok := encryptedKeys[preferredKey]; ok {
			key, err := decryptWithGPG(encKey)
			if err == nil {
				return key, preferredKey, nil
			}
		}
	}

	// Try all available keys
	for keyName, encKey := range encryptedKeys {
		key, err := decryptWithGPG(encKey)
		if err == nil {
			return key, keyName, nil
		}
	}

	return nil, "", fmt.Errorf("no suitable GPG key found for decryption")
}

func decryptWithGPG(armoredData string) ([]byte, error) {
	// Try to decrypt with GPG
	md, err := openpgp.ReadMessage(strings.NewReader(armoredData), nil, nil, nil)
	if err != nil {
		return nil, err
	}
	
	plaintext, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

func decryptAESGCM(ciphertext, key, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}