#!/bin/bash

echo "Building binaries..."
make build

SESSION_NAME="test-session-$RANDOM"
SESSION_NAME_2="${SESSION_NAME}-B"

UUID=$(./bin/ads new "$SESSION_NAME" | grep UUID | awk '{print $2}')
UUID_2=$(./bin/ads new "$SESSION_NAME_2" | grep UUID | awk '{print $2}')
echo "Created session 1: $UUID"
echo "Created session 2: $UUID_2"

echo "Piping commands to ads-shell..."
echo "sleep 1 && echo 'hello from ads-shell' && sleep 1 && exit" | ./bin/ads-shell --session "$UUID"

echo "Checking DB (search --json)..."
JSON_OUT=$(./bin/ads search "hello" --json)
if echo "$JSON_OUT" | grep -q '"SessionName":'; then
    echo "Verification passed (JSON format detected)."
    
    echo "Testing glob session deletion..."
    if ! ./bin/ads delete "$SESSION_NAME*"; then
        echo "FAILURE: Could not delete sessions via glob pattern."
        exit 1
    fi

    echo "Checking if DB files were physically deleted..."
    DB_FILE="$HOME/.local/share/ads/sessions/$UUID.db"
    DB_FILE_2="$HOME/.local/share/ads/sessions/$UUID_2.db"
    if [ -f "$DB_FILE" ] || [ -f "$DB_FILE_2" ]; then
        echo "FAILURE: DB files still exist after glob deletion!"
        exit 1
    fi

    echo "All tests passed successfully."
else
    echo "FAILURE: Output not found in JSON format."
    echo "Got: $JSON_OUT"
    exit 1
fi
