#!/bin/bash
set -e

echo "=== Starting Full E2E Autonomous Pipeline ==="

echo "1. Building new RPM Release locally..."
make rpm

echo "2. Installing RPM autonomously via host-exec (zero intervention)..."
./scripts/host-exec.sh "cd /home/dada/Documents/gitlab.dada/advanced-dada-system && ./upgrade.sh"

echo "3. Provisioning a new session natively on the host..."
SESSION_NAME="e2e-auto-$(date +%s)"
./scripts/host-exec.sh "/usr/bin/ads new $SESSION_NAME"

echo "4. Injecting test payload into live session natively on the host..."
# We send the special hidden diagnostic command we compiled into the new RPM release
TEST_PAYLOAD="PIPELINE-TEST-$(date +%s)"
# The exit command forces the session to close after the diagnostic runs
./scripts/host-exec.sh "echo -e 'ads test-diagnostic $TEST_PAYLOAD\nexit\n' | /usr/bin/ads run $SESSION_NAME"

echo "5. Verifying DB Persistence natively on the host..."
# Give the recorder a second to flush to SQLite
sleep 2

# We use the host's ads search command to federated-query the DBs for our payload
SEARCH_RESULT=$(./scripts/host-exec.sh "/usr/bin/ads search $TEST_PAYLOAD")

echo "=== Search Verification ==="
echo "$SEARCH_RESULT"

if echo "$SEARCH_RESULT" | grep -Fq "[ADS-DIAGNOSTIC-E2E] $TEST_PAYLOAD"; then
    echo "✅ E2E PIPELINE SUCCESS: The host successfully recorded and retrieved the diagnostic execution!"
    exit 0
else
    echo "❌ E2E PIPELINE FAILED: The diagnostic execution was not found in the host database."
    exit 1
fi
