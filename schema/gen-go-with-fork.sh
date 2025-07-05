#!/bin/bash
set -euo pipefail

# Configuration
QUICKTYPE_REPO="https://github.com/haya14busa/quicktype.git"
QUICKTYPE_BRANCH="fix-unevaluated-properties-support"
QUICKTYPE_DIR="/tmp/quicktype-fork"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "🔧 Setting up forked quicktype..."

# Clone or update the fork
if [ -d "$QUICKTYPE_DIR" ]; then
    echo "📦 Updating existing quicktype fork..."
    cd "$QUICKTYPE_DIR"
    git fetch origin
    git checkout "$QUICKTYPE_BRANCH"
    git pull origin "$QUICKTYPE_BRANCH"
else
    echo "📦 Cloning quicktype fork..."
    git clone "$QUICKTYPE_REPO" --branch "$QUICKTYPE_BRANCH" "$QUICKTYPE_DIR"
    cd "$QUICKTYPE_DIR"
fi

# Check if already built
if [ ! -f "$QUICKTYPE_DIR/dist/index.js" ]; then
    echo "🔨 Building quicktype (this may take a while on first run)..."
    npm install
    npm run build
else
    echo "✅ Using existing build"
fi

# Add quicktypePropertyOrder to JSON Schema
echo "📝 Adding quicktypePropertyOrder to JSON Schema..."
cd "$SCRIPT_DIR"

# Check if deno is installed
if ! command -v deno &> /dev/null; then
    echo "❌ Deno is not installed. Please install Deno first."
    echo "Visit: https://deno.land/manual/getting_started/installation"
    exit 1
fi

deno run --allow-read --allow-write add-quicktype-property-order.ts

# Generate Go structs
echo "🚀 Generating Go structs..."
node "$QUICKTYPE_DIR/dist/index.js" \
    --src "output/@typespec/json-schema/InstallSpec.json" \
    --src-lang schema \
    --lang go \
    --package spec \
    -o "../pkg/spec/generated.go" \
    --all-properties-optional \
    --top-level InstallSpec

echo "✅ Go structs generated successfully!"
echo "📄 Output: $SCRIPT_DIR/../pkg/spec/generated.go"

# Format the generated Go code
echo "🎨 Formatting generated Go code..."
gofmt -w ../pkg/spec/generated.go

# Show the EmbeddedChecksums type to verify it's correct
echo ""
echo "🔍 Checking EmbeddedChecksums type:"
grep -A2 "EmbeddedChecksums" ../pkg/spec/generated.go | grep -v "^--" || true