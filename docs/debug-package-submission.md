# Debug Package Submission

The `lncli submitdebugpackage` command provides a secure and convenient way to collect, encrypt, and share debug information with Lightning Labs support team.

## Overview

When troubleshooting issues with your Lightning Network node, providing comprehensive debug information is crucial. The `submitdebugpackage` command automates this process by:

1. Collecting relevant debug information (config, logs, network info)
2. Compressing the data for efficient transmission
3. Encrypting it using industry-standard encryption
4. Uploading to a paste service or saving locally
5. Providing a shareable URL/path for support tickets

## Security Model

The command uses a hybrid encryption approach to ensure your debug information remains confidential:

### Encryption Process

1. **Symmetric Encryption**: Your debug data is encrypted using AES-256-GCM with a randomly generated key
2. **Asymmetric Key Wrapping**: The AES key is then encrypted separately for each Lightning Labs developer using their GPG public keys
3. **Access Control**: Only Lightning Labs developers with corresponding private keys can decrypt the package

This ensures that even if the encrypted package is uploaded to a public service, only authorized Lightning Labs personnel can access the contents.

## Usage

### Basic Usage

Submit debug information with automatic service selection:

```bash
# With GitHub token (creates private gist)
export GITHUB_TOKEN=your_token_here
lncli submitdebugpackage

# Without token (uses dpaste.com)
lncli submitdebugpackage
```

### Including Additional Information

Add more context to help with troubleshooting:

```bash
# Include peer information
lncli submitdebugpackage --peers

# Include channel information
lncli submitdebugpackage --channels

# Include on-chain transaction information
lncli submitdebugpackage --onchain

# Include everything
lncli submitdebugpackage --peers --channels --onchain
```

### Specifying Upload Service

Choose where to upload the encrypted package:

```bash
# Force GitHub Gist (requires token)
lncli submitdebugpackage --service=github --github-token=your_token

# Force dpaste.com (anonymous, 7-day expiry)
lncli submitdebugpackage --service=dpaste

# Save to local file
lncli submitdebugpackage --service=file --output-file=debug.json
```

### Dry Run

Test the collection and encryption without uploading:

```bash
lncli submitdebugpackage --dry-run
```

## Upload Services

### GitHub Gist
- **Pros**: Private gists, permanent storage, direct GitHub integration
- **Cons**: Requires GitHub token
- **How to get token**: 
  1. Go to GitHub Settings → Developer settings → Personal access tokens
  2. Generate new token with `gist` scope
  3. Set as `GITHUB_TOKEN` environment variable or use `--github-token` flag

### dpaste.com
- **Pros**: No authentication required, simple and fast
- **Cons**: 7-day expiry, size limitations
- **Use case**: Quick debugging sessions, temporary issues

### Local File
- **Pros**: Complete control, no size limits, no internet required
- **Cons**: Manual sharing required
- **Use case**: Sensitive environments, large debug packages

## Information Collected

By default, the command collects:
- Node information (`getinfo`)
- Configuration (with sensitive values masked)
- Recent log entries (last 10MB if logs are larger)
- Network information

With additional flags:
- `--peers`: Peer connections and errors
- `--channels`: Open, pending, and closed channels
- `--onchain`: On-chain transactions and UTXOs

## For Lightning Labs Developers

### Decrypting Packages

Two tools are provided for decryption:

#### Go Decryption Tool

Full-featured decryption with automatic URL handling:

```bash
# From GitHub Gist
go run cmd/lncli-decrypt-debug/main.go -input https://gist.github.com/user/abc123

# From dpaste
go run cmd/lncli-decrypt-debug/main.go -input https://dpaste.com/ABCDEF

# From local file
go run cmd/lncli-decrypt-debug/main.go -input debug-package.json

# Specify output file
go run cmd/lncli-decrypt-debug/main.go -input <url> -output decrypted.json
```

#### Shell Script Helper

Quick inspection tool for package metadata:

```bash
# Shows package info and available keys
./scripts/decrypt-debug-package.sh <url-or-file>
```

### Required GPG Keys

Your GPG private key must correspond to one of the public keys in `scripts/keys/`. The package will show which keys can decrypt it.

## Troubleshooting

### "Package too large for dpaste"
- Use GitHub Gist with a token, or save to local file
- Consider excluding some information (omit `--peers`, `--channels`, etc.)

### "No GPG keys found"
- Ensure you're running from the lnd repository root
- Check that `scripts/keys/` directory exists and contains `.asc` files

### "Unable to decrypt" (for developers)
- Verify your GPG key is in `scripts/keys/`
- Ensure you have the corresponding private key
- Check that your key hasn't expired

## Privacy Considerations

- **Logs are truncated**: Only the last 10MB of logs are included
- **Private gists**: When using GitHub with a token, gists are private
- **Temporary storage**: dpaste.com links expire after 7 days
- **Local control**: Use `--service=file` for complete control over the data

## Examples

### For Users Reporting Issues

```bash
# Quick submission for a channel issue
lncli submitdebugpackage --channels

# Comprehensive package for complex issues
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
lncli submitdebugpackage --peers --channels --onchain

# Save locally for manual review before sharing
lncli submitdebugpackage --service=file --output-file=debug-$(date +%Y%m%d).json
```

### For Lightning Labs Developers

```bash
# User provides: "I uploaded debug info to https://gist.github.com/anonymous/a1b2c3d4"
go run cmd/lncli-decrypt-debug/main.go -input https://gist.github.com/anonymous/a1b2c3d4

# Output will be saved to debug-package-decrypted.json
# View specific sections:
jq '.config' debug-package-decrypted.json      # View configuration
jq '.log[-100:]' debug-package-decrypted.json  # View last 100 log lines
```

## Implementation Details

The feature is implemented across several files:

- `cmd/commands/cmd_debug.go`: Main command implementation
- `cmd/lncli-decrypt-debug/main.go`: Decryption tool
- `scripts/decrypt-debug-package.sh`: Shell helper for decryption
- `scripts/keys/*.asc`: Lightning Labs developer GPG public keys

The encryption uses:
- AES-256-GCM for symmetric encryption
- GPG/OpenPGP for key wrapping
- Brotli compression for size reduction
- Base64 encoding for JSON compatibility