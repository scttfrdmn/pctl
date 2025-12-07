#!/bin/bash
# Remove petal testing SSH key from AWS regions

set -e

KEY_NAME="petal-testing-key"
REGIONS=("us-west-2" "us-east-1" "us-east-2" "eu-west-1")

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🗑️  Removing petal testing key from AWS regions..."
echo ""

# Remove from each region
for region in "${REGIONS[@]}"; do
    echo -n "  $region ... "

    # Check if key exists
    if aws ec2 describe-key-pairs --key-names "$KEY_NAME" --region "$region" >/dev/null 2>&1; then
        # Delete the key
        if aws ec2 delete-key-pair --key-name "$KEY_NAME" --region "$region" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ deleted${NC}"
        else
            echo -e "${RED}❌ failed${NC}"
        fi
    else
        echo -e "${YELLOW}not found (skipping)${NC}"
    fi
done

echo ""
echo "✅ Cleanup complete!"
