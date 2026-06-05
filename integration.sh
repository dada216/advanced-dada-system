#!/bin/bash

echo "Building binaries..."
make build

SESSION_NAME="test-session-$RANDOM-$RANDOM"
UUID=$(./bin/ads new "$SESSION_NAME" | grep UUID | awk '{print $2}')
echo "Created session: $UUID"

echo "Starting detached tmux session..."
tmux new-session -d -s "$UUID" "sleep 1 && echo 'hello from tmux' && sleep 2"

echo "Piping pane to ads-recorder..."
tmux pipe-pane -t "$UUID" "$PWD/bin/ads-recorder --session $UUID 2>> /tmp/ads-recorder-$UUID.log"

echo "Waiting for session to finish..."
sleep 3

echo "Checking DB..."
if go run -tags sqlite_fts5 test/verify.go "$UUID"; then
    echo "Verification passed."
else
    echo "FAILURE: Output not found in DB."
    echo "Recorder log:"
    cat /tmp/ads-recorder-$UUID.log
    exit 1
fi
