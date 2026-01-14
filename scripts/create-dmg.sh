#!/bin/bash
# create-dmg.sh - Create DMG files for macOS distribution
# Usage: ./scripts/create-dmg.sh [version]
# Run after goreleaser to create DMG files from the built binaries

set -e

VERSION="${1:-dev}"
DIST_DIR="dist"
MAX_RETRIES=3

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[DMG]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    exit 1
}

# Check if running on macOS
if [[ "$(uname)" != "Darwin" ]]; then
    error "DMG creation requires macOS (hdiutil)"
fi

# Check if dist directory exists
if [[ ! -d "$DIST_DIR" ]]; then
    error "dist/ directory not found. Run 'goreleaser release --snapshot --clean' first"
fi

# Retry wrapper for hdiutil (handles intermittent "Resource busy" errors)
hdiutil_with_retry() {
    local attempt=1
    while [[ $attempt -le $MAX_RETRIES ]]; do
        if hdiutil "$@" 2>&1; then
            return 0
        fi
        warn "hdiutil failed (attempt $attempt/$MAX_RETRIES), retrying in 5s..."
        sleep 5
        ((attempt++))
    done
    error "hdiutil failed after $MAX_RETRIES attempts"
}

create_dmg() {
    local arch="$1"
    local label="$2"
    local binary_path=""

    # Find the binary in dist directory
    # GoReleaser creates directories like: nylas_darwin_arm64_v8.0/nylas
    for dir in "$DIST_DIR"/nylas_darwin_${arch}*/; do
        if [[ -f "${dir}nylas" ]]; then
            binary_path="${dir}nylas"
            break
        fi
    done

    if [[ -z "$binary_path" || ! -f "$binary_path" ]]; then
        log "Skipping $label - binary not found for darwin/$arch"
        return
    fi

    local dmg_name="Nylas-CLI-${VERSION}-${label}.dmg"
    local tmp_dir=$(mktemp -d)
    local volume_name="Nylas CLI ${VERSION}"

    log "Creating $dmg_name..."

    # Copy binary to temp directory
    cp "$binary_path" "$tmp_dir/nylas"
    chmod +x "$tmp_dir/nylas"

    # Copy README if exists
    if [[ -f "README.md" ]]; then
        cp "README.md" "$tmp_dir/"
    fi

    # Create DMG with retry logic
    hdiutil_with_retry create \
        -volname "$volume_name" \
        -srcfolder "$tmp_dir" \
        -ov \
        -format UDZO \
        "$DIST_DIR/$dmg_name"

    # Cleanup
    rm -rf "$tmp_dir"

    log "Created: $DIST_DIR/$dmg_name"
}

log "Creating DMG files for version $VERSION"

# Create DMGs for both architectures
create_dmg "arm64" "apple-silicon"
create_dmg "amd64" "intel"

log "Done! DMG files created in $DIST_DIR/"
