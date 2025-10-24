#!/bin/bash

# https://www2.correios.com.br/sistemas/edne/

# Configuration
LOGIN_URL='https://www2.correios.com.br/sistemas/edne/login.cfm'
DOWNLOAD_URL='https://www2.correios.com.br/sistemas/edne/download/eDNE_Basico.zip'
USERNAME="${1:-}"
PASSWORD="${2:-}"

if [ -z "$USERNAME" ] || [ -z "$PASSWORD" ]; then
    echo "Usage: $0 <username> <password>"
    exit 1
fi
COOKIE_FILE='cookies.txt'
OUTPUT_FILE='eDNE_Basico.zip'

rm -rf ./tmp_dne/
rm -rf ./dne/

echo "Attempting download with curl..."
curl -c "$COOKIE_FILE" -d "tx_codigo=$USERNAME&tx_senha=$PASSWORD" "$LOGIN_URL" && \
curl -b "$COOKIE_FILE" -o "$OUTPUT_FILE" "$DOWNLOAD_URL"

rm -f "$COOKIE_FILE"

echo "Download completed: $OUTPUT_FILE"

unzip -o "$OUTPUT_FILE" -d ./tmp_dne/

unzip -o ./tmp_dne/eDNE_Basico.zip -d ./tmp_dne/
unzip -o ./tmp_dne/eDNE_Basico*.zip -d ./tmp_dne/

mv ./tmp_dne/Delimitado ./dne

ls -lh ./tmp_dne/
