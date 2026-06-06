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
    
    echo "Testing Zsh integration..."
    ZSH_SESSION="test-session-zsh-$RANDOM"
    ZSH_UUID=$(./bin/ads new "$ZSH_SESSION" | grep UUID | awk '{print $2}')
    echo "Created Zsh session: $ZSH_UUID"
    
    # Run ads-shell explicitly telling it to use zsh
    echo -e "echo 'zsh input capture test'\nexit\n" | ./bin/ads-shell --session "$ZSH_UUID" --shell "zsh"
    
    ZSH_JSON_OUT=$(./bin/ads search "zsh input capture test" --json)
    if echo "$ZSH_JSON_OUT" | grep -q '"SessionName":'; then
        echo "Zsh test passed (search found captured content)."
    else
        echo "FAILURE: Zsh input/output not captured."
        echo "Got: $ZSH_JSON_OUT"
        exit 1
    fi
    
    echo "Verifying io_stream direction capturing..."
    cat << 'EOF' > verify_db.py
import sqlite3, sys
db = sys.argv[1]
conn = sqlite3.connect(db)
cursor = conn.cursor()
cursor.execute("SELECT COUNT(*) FROM io_stream WHERE direction=0;")
in_count = cursor.fetchone()[0]
cursor.execute("SELECT COUNT(*) FROM io_stream WHERE direction=1;")
out_count = cursor.fetchone()[0]
if in_count > 0 and out_count > 0:
    print("Verified direction tracking (in: {}, out: {})".format(in_count, out_count))
    sys.exit(0)
else:
    print("Missing direction data (in: {}, out: {})".format(in_count, out_count))
    sys.exit(1)
EOF
    
    if python3 verify_db.py "$HOME/.local/share/ads/sessions/$ZSH_UUID.db"; then
        echo "Direction verification passed."
    else
        echo "FAILURE: Direction tracking verification failed."
        exit 1
    fi
    rm verify_db.py
    
    echo "Testing glob session deletion..."
    if ! ./bin/ads delete "$SESSION_NAME*"; then
        echo "FAILURE: Could not delete sessions via glob pattern."
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
