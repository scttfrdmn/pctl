#!/bin/bash
# Upload petal testing SSH key to AWS regions

set -e

KEY_NAME="petal-testing-key"
KEY_FILE=".aws-keys/petal-testing-key.pub"
REGIONS=("us-west-2" "us-east-1" "us-east-2" "eu-west-1")
AWS_PROFILE="${AWS_PROFILE:-aws}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔑 Uploading petal testing key to AWS regions..."
echo ""

# Check if key file exists
if [ ! -f "$KEY_FILE" ]; then
    echo -e "${RED}❌ Key file not found: $KEY_FILE${NC}"
    echo "Run this from the project root directory"
    exit 1
fi

# Upload to each region
for region in "${REGIONS[@]}"; do
    echo -n "  $region ... "

    # Check if key already exists
    if AWS_PROFILE="$AWS_PROFILE" aws ec2 describe-key-pairs --key-names "$KEY_NAME" --region "$region" >/dev/null 2>&1; then
        echo -e "${YELLOW}exists (skipping)${NC}"
    else
        # Import the key
        if AWS_PROFILE="$AWS_PROFILE" aws ec2 import-key-pair \
            --key-name "$KEY_NAME" \
            --public-key-material "fileb://$KEY_FILE" \
            --region "$region" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ uploaded${NC}"
        else
            echo -e "${RED}❌ failed${NC}"
        fi
    fi
done

echo ""
echo "✅ Key upload complete!"
echo ""
echo "Usage:"
echo "  petal create --seed my-seed.yaml --key-name $KEY_NAME"
