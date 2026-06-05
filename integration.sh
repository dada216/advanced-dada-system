#!/bin/bash

echo "Building binaries..."
make build

SESSION_NAME="test-session-$RANDOM-$RANDOM"
UUID=$(./bin/ads new "$SESSION_NAME" | grep UUID | awk '{print $2}')
echo "Created session: $UUID"

echo "Starting detached tmux session..."
tmux new-session -d -s "$UUID" "echo 'hello from tmux' && sleep 2"

echo "Piping pane to ads-recorder..."
tmux pipe-pane -t "$UUID" "$PWD/bin/ads-recorder --session $UUID 2>> /tmp/ads-recorder-$UUID.log"

echo "Waiting for session to finish..."
sleep 3

echo "Checking DB..."
sqlite3 ~/.local/share/ads/sessions/$UUID.db "SELECT text FROM fts_index;" > /tmp/out.log
if grep -q "hello from tmux" /tmp/out.log; then
    echo "SUCCESS: Found output in SQLite DB!"
else
    echo "FAILURE: Output not found in DB."
    echo "SQLite contents:"
    cat /tmp/out.log
    echo "Recorder log:"
    cat /tmp/ads-recorder-$UUID.log
    exit 1
fi
