#!/bin/bash

echo "Building binaries..."
make build

SESSION_NAME="test-session-$RANDOM-$RANDOM"
UUID=$(./bin/ads new "$SESSION_NAME" | grep UUID | awk '{print $2}')
echo "Created session: $UUID"

echo "Piping commands to ads-shell..."
echo "sleep 1 && echo 'hello from ads-shell' && sleep 1 && exit" | ./bin/ads-shell --session "$UUID"

echo "Checking DB..."
if go run -mod=vendor -tags sqlite_fts5 test/verify.go "$UUID"; then
    echo "Verification passed."
    
    echo "Testing session deletion..."
    if ! ./bin/ads delete "$SESSION_NAME"; then
        echo "FAILURE: Could not delete session '$SESSION_NAME' via ads CLI."
        exit 1
    fi

    echo "Checking if DB file was physically deleted..."
    DB_FILE="$HOME/.local/share/ads/sessions/$UUID.db"
    if [ -f "$DB_FILE" ]; then
        echo "FAILURE: DB file $DB_FILE still exists after deletion!"
        exit 1
    fi

    echo "All tests passed successfully."
else
    echo "FAILURE: Output not found in DB."
    exit 1
fi
