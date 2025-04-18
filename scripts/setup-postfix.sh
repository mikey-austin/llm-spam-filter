#!/bin/bash
set -e

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root"
  exit 1
fi

# Backup original master.cf
echo "Backing up original Postfix configuration..."
cp /etc/postfix/master.cf /etc/postfix/master.cf.bak.$(date +%Y%m%d%H%M%S)

# Check if the content filter entries already exist
if grep -q "llm-spam-filter" /etc/postfix/master.cf; then
  echo "LLM spam filter already configured in master.cf"
else
  echo "Adding LLM spam filter configuration to master.cf..."
  cat >> /etc/postfix/master.cf << 'EOF'

# LLM Spam Filter Configuration
llm-spam-filter unix - n n - - pipe
  flags=Rq user=nobody null_sender=
  argv=/usr/local/bin/llm-spam-filter

# Postfix service for receiving mail from the content filter
127.0.0.1:10026 inet n - n - - smtpd
  -o content_filter=
  -o receive_override_options=no_unknown_recipient_checks,no_header_body_checks
  -o smtpd_helo_restrictions=
  -o smtpd_client_restrictions=
  -o smtpd_sender_restrictions=
  -o smtpd_recipient_restrictions=permit_mynetworks,reject
  -o mynetworks=127.0.0.0/8
  -o smtpd_authorized_xforward_hosts=127.0.0.0/8
EOF
fi

# Update main.cf to use the content filter
if grep -q "content_filter = llm-spam-filter" /etc/postfix/main.cf; then
  echo "Content filter already configured in main.cf"
else
  echo "Configuring content filter in main.cf..."
  echo "# LLM Spam Filter" >> /etc/postfix/main.cf
  echo "content_filter = llm-spam-filter:dummy" >> /etc/postfix/main.cf
fi

echo "Configuration complete. Please restart Postfix with: systemctl restart postfix"
