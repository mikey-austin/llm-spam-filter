#!/bin/bash
# Setup script for Postfix integration

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
cp /etc/postfix/master.cf /etc/postfix/master.cf.bak.$TIMESTAMP

# Copy configuration files
echo "Copying configuration files..."
cat configs/postfix/main.cf >> /etc/postfix/main.cf
cat configs/postfix/master.cf >> /etc/postfix/master.cf
cp configs/postfix/transport /etc/postfix/transport

# Generate transport map
echo "Generating transport map..."
postmap /etc/postfix/transport

# Restart Postfix
echo "Restarting Postfix..."
systemctl restart postfix

echo "Postfix configuration completed successfully!"
echo "The LLM Spam Filter is now integrated with Postfix."
echo "Make sure the LLM Spam Filter service is running."
