#!/bin/bash
# host-exec.sh
# Wraps systemd-run to execute commands natively on the host system from within the container.
# All commands are rigorously audited to docs/security/host_exec.log for transparency.

if [ "$#" -eq 0 ]; then
    echo "Usage: $0 <command>"
    exit 1
fi

COMMAND="$*"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
# Log path relative to the script location
LOG_FILE="$(dirname "$0")/../docs/security/host_exec.log"

mkdir -p "$(dirname "$LOG_FILE")"

# Write to the audit log in a clear, human-readable format
echo "[${TIMESTAMP}] HOST EXECUTION AUDIT EVENT" >> "$LOG_FILE"
echo "  COMMAND   : ${COMMAND}" >> "$LOG_FILE"

echo "[host-exec] Requesting host execution via systemd-run (D-Bus)..."
# Executing natively on the host user session via systemd D-Bus
systemd-run --user --wait --pipe --quiet bash -c "${COMMAND}"
EXIT_CODE=$?

echo "  EXIT_CODE : ${EXIT_CODE}" >> "$LOG_FILE"
echo "------------------------------------------------------------" >> "$LOG_FILE"

exit $EXIT_CODE
