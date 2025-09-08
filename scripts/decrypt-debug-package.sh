#!/bin/bash

# Script to decrypt debug packages submitted via lncli submitdebugpackage
# Usage: ./decrypt-debug-package.sh <url-or-json-file> [gpg-key-id]

set -e

if [ $# -lt 1 ]; then
    echo "Usage: $0 <url-or-json-file> [gpg-key-id]"
    echo ""
    echo "Examples:"
    echo "  $0 https://gist.github.com/user/abc123"
    echo "  $0 https://dpaste.com/ABCDEF"
    echo "  $0 debug-package.json"
    echo "  $0 debug-package.json my-key@example.com"
    exit 1
fi

INPUT="$1"
GPG_KEY="${2:-}"

# Check if input is a URL or file
if [[ "$INPUT" == https://gist.github.com/* ]]; then
    echo "Fetching from GitHub Gist..."
    # Extract gist ID from URL
    GIST_ID=$(echo "$INPUT" | sed 's|.*/||' | cut -d'#' -f1)
    
    # Download the raw JSON file
    JSON_FILE=$(mktemp)
    curl -s "https://gist.githubusercontent.com/raw/$GIST_ID" > "$JSON_FILE"
    
elif [[ "$INPUT" == *dpaste.com/* ]]; then
    echo "Fetching from dpaste..."
    # Add .json extension if not present
    URL="$INPUT"
    if [[ ! "$URL" == *.json ]]; then
        URL="${URL}.json"
    fi
    
    JSON_FILE=$(mktemp)
    curl -s "$URL" > "$JSON_FILE"
    
else
    JSON_FILE="$INPUT"
fi

if [ ! -f "$JSON_FILE" ]; then
    echo "Error: File $JSON_FILE not found"
    exit 1
fi

echo "Parsing encrypted package..."

# Extract components from JSON
ENCRYPTED_DATA=$(jq -r '.encrypted_data' "$JSON_FILE")
NONCE=$(jq -r '.nonce' "$JSON_FILE")
TIMESTAMP=$(jq -r '.timestamp' "$JSON_FILE")
VERSION=$(jq -r '.version' "$JSON_FILE")

# Find which key we can use
echo "Finding available GPG key..."
AVAILABLE_KEYS=$(jq -r '.encrypted_keys | keys[]' "$JSON_FILE")

echo "Available keys in package:"
echo "$AVAILABLE_KEYS"

# Try to find a matching key
SELECTED_KEY=""
for KEY in $AVAILABLE_KEYS; do
    # Check if we have the private key for this
    if gpg --list-secret-keys 2>/dev/null | grep -qi "$KEY"; then
        SELECTED_KEY="$KEY"
        echo "Found matching key: $KEY"
        break
    fi
done

if [ -z "$SELECTED_KEY" ]; then
    echo ""
    echo "Error: No matching GPG private key found"
    echo "You need the private key for one of the above identities"
    echo ""
    echo "To decrypt this package, use the Go decryption tool:"
    echo "  go run cmd/lncli-decrypt-debug/main.go -input $INPUT"
    exit 1
fi

echo ""
echo "Package info:"
echo "  Timestamp: $(date -r $TIMESTAMP 2>/dev/null || date -d @$TIMESTAMP)"
echo "  Version: $VERSION"
echo "  Selected key: $SELECTED_KEY"
echo ""
echo "Note: To fully decrypt this package, use the Go decryption tool:"
echo "  go run cmd/lncli-decrypt-debug/main.go -input $INPUT"

# Clean up temp file if we downloaded it
if [[ "$INPUT" == https://* ]]; then
    rm -f "$JSON_FILE"
fi