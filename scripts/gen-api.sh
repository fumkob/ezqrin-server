#!/bin/bash

# gen-api.sh - Generate API code from OpenAPI specification
# This script generates type-safe Go code from the OpenAPI spec using oapi-codegen

set -e  # Exit on error
set -u  # Exit on undefined variable

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory (for relative paths)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration
OPENAPI_SPEC="$PROJECT_ROOT/api/openapi.yaml"
CONFIG_FILE="$PROJECT_ROOT/config/oapi-codegen.yaml"
OUTPUT_DIR="$PROJECT_ROOT/internal/interface/api/generated"
OUTPUT_FILE="$OUTPUT_DIR/api.gen.go"

echo -e "${GREEN}=== ezQRin API Code Generation ===${NC}"
echo ""

# Check if OpenAPI spec exists
if [ ! -f "$OPENAPI_SPEC" ]; then
    echo -e "${RED}ERROR: OpenAPI specification not found at $OPENAPI_SPEC${NC}"
    exit 1
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}ERROR: oapi-codegen configuration not found at $CONFIG_FILE${NC}"
    exit 1
fi

# Check if oapi-codegen is installed
if ! command -v oapi-codegen &> /dev/null; then
    echo -e "${RED}ERROR: oapi-codegen not found in PATH${NC}"
    echo -e "${RED}Please ensure you're running in the DevContainer, or install manually:${NC}"
    echo -e "${RED}  go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1${NC}"
    exit 1
fi

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

echo "Configuration:"
echo "  OpenAPI Spec: $OPENAPI_SPEC"
echo "  Config File:  $CONFIG_FILE"
echo "  Output:       $OUTPUT_FILE"
echo ""

# Bundle multi-file OpenAPI spec into single file
BUNDLED_SPEC="$PROJECT_ROOT/.tmp/openapi-bundled.yaml"
mkdir -p "$PROJECT_ROOT/.tmp"

echo -e "${YELLOW}Bundling multi-file OpenAPI specification...${NC}"

# Check if swagger-cli is available
if ! command -v swagger-cli &> /dev/null; then
    echo -e "${RED}ERROR: swagger-cli not found in PATH${NC}"
    echo -e "${RED}Please ensure you're running in the DevContainer, or install manually:${NC}"
    echo -e "${RED}  npm install -g @apidevtools/swagger-cli@4.0.4${NC}"
    exit 1
fi

# Bundle using swagger-cli
swagger-cli bundle "$OPENAPI_SPEC" -o "$BUNDLED_SPEC" -t yaml

if [ $? -ne 0 ]; then
    echo -e "${RED}ERROR: Failed to bundle OpenAPI specification${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Specification bundled successfully${NC}"

# Validate bundled spec
echo -e "${YELLOW}Validating bundled OpenAPI specification...${NC}"
if command -v openapi-generator-cli &> /dev/null; then
    openapi-generator-cli validate -i "$BUNDLED_SPEC" || echo -e "${YELLOW}Note: validation warnings can be ignored if generation succeeds${NC}"
else
    echo -e "${YELLOW}Note: Install openapi-generator-cli for spec validation${NC}"
fi

# Generate code from bundled spec
echo -e "${YELLOW}Generating API code...${NC}"

oapi-codegen \
    -config="$CONFIG_FILE" \
    -package=generated \
    -o="$OUTPUT_FILE" \
    "$BUNDLED_SPEC"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Code generation successful!${NC}"
    echo ""
    echo "Generated files:"
    echo "  - $OUTPUT_FILE"
    echo ""

    # Format generated code
    echo -e "${YELLOW}Formatting generated code...${NC}"
    gofmt -s -w "$OUTPUT_FILE"
    echo -e "${GREEN}✓ Code formatted${NC}"

    # Run go mod tidy to ensure dependencies are correct
    echo -e "${YELLOW}Running go mod tidy...${NC}"
    cd "$PROJECT_ROOT" && go mod tidy
    echo -e "${GREEN}✓ Dependencies updated${NC}"

    echo ""
    echo -e "${GREEN}=== Code Generation Complete ===${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Review generated code: internal/interface/api/generated/api.gen.go"
    echo "  2. Implement server interfaces in internal/interface/api/handler/"
    echo "  3. Run tests: make test"
else
    echo -e "${RED}✗ Code generation failed${NC}"
    exit 1
fi
