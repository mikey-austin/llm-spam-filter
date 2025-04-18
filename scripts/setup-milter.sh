#!/bin/bash
# Setup script for Milter integration with Postfix

set -e

# Check if running as root
if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root" >&2
    exit 1
fi

# Check if Postfix is installed
if ! command -v postfix &> /dev/null; then
    echo "Postfix is not installed. Please install it first."
    exit 1
fi

# Create backup of original configuration
TIMESTAMP=$(date +%Y%m%d%H%M%S)
echo "Creating backup of Postfix configuration..."
cp /etc/postfix/main.cf /etc/postfix/main.cf.bak.$TIMESTAMP

# Copy configuration files
echo "Copying configuration files..."
cat configs/postfix/milter-integration.cf >> /etc/postfix/main.cf

# Update the configuration file to use Milter
echo "Updating LLM Spam Filter configuration..."
sed -i 's/filter_type: "postfix"/filter_type: "milter"/' configs/config.yaml

# Restart Postfix
echo "Restarting Postfix..."
systemctl restart postfix

echo "Milter configuration completed successfully!"
echo "The LLM Spam Filter is now integrated with Postfix as a Milter."
echo "Make sure the LLM Spam Filter service is running."
