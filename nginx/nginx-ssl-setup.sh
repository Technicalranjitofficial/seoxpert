#!/usr/bin/env bash
# nginx-ssl-setup.sh
# Run this on the Oracle server as ubuntu@ to install Nginx + Let's Encrypt SSL.
# Usage: bash nginx-ssl-setup.sh seoxpert.io your@email.com

set -euo pipefail

DOMAIN="${1:-seoxpert.io}"
EMAIL="${2:-admin@seoxpert.io}"
API_DOMAIN="api.${DOMAIN}"

echo "=== Installing Nginx + Certbot ==="
sudo apt-get update -q
sudo apt-get install -y nginx certbot python3-certbot-nginx

echo "=== Copying Nginx config ==="
sudo cp /home/ubuntu/seoxpert/nginx/conf.d/seoxpert.conf /etc/nginx/conf.d/seoxpert.conf

# Disable default site if present
sudo rm -f /etc/nginx/sites-enabled/default

# Temp config for ACME challenge (before certs exist)
sudo tee /etc/nginx/conf.d/acme-temp.conf > /dev/null << 'EOF'
server {
    listen 80 default_server;
    server_name _;
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }
    location / { return 444; }
}
EOF

sudo mkdir -p /var/www/certbot
sudo nginx -t && sudo systemctl reload nginx

echo "=== Obtaining SSL certificates ==="
sudo certbot certonly --webroot \
  -w /var/www/certbot \
  -d "${DOMAIN}" -d "www.${DOMAIN}" \
  --email "${EMAIL}" \
  --agree-tos \
  --non-interactive

sudo certbot certonly --webroot \
  -w /var/www/certbot \
  -d "${API_DOMAIN}" \
  --email "${EMAIL}" \
  --agree-tos \
  --non-interactive

echo "=== Removing temp config and loading final config ==="
sudo rm /etc/nginx/conf.d/acme-temp.conf
sudo nginx -t && sudo systemctl reload nginx

echo "=== Setting up auto-renewal ==="
(crontab -l 2>/dev/null; echo "0 3 * * * certbot renew --quiet && systemctl reload nginx") | crontab -

echo ""
echo "Done! Test:"
echo "  curl -I https://${DOMAIN}"
echo "  curl -I https://${API_DOMAIN}/health"
