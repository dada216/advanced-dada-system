#!/bin/bash

# Exit on error
set -e

# Parse optional tags (for future use)
TAGS=$@

echo "=== ADS Upgrade Script ==="
if [ -n "$TAGS" ]; then
    echo "Running with tags: $TAGS"
fi

# Ask for sudo credentials once upfront and keep them alive
echo "Requesting sudo privileges for installation tasks..."
sudo -v
# Keep-alive: update existing sudo time stamp until script has finished
while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null &

# Ensure rpmbuild is installed
if ! command -v rpmbuild &> /dev/null; then
    echo "rpmbuild not found. Attempting to install rpm-build package..."
    if command -v dnf &> /dev/null; then
        sudo dnf install -y rpm-build
    elif command -v yum &> /dev/null; then
        sudo yum install -y rpm-build
    elif command -v zypper &> /dev/null; then
        sudo zypper install -y rpm-build
    else
        echo "Error: Could not find package manager to install rpm-build. Please install it manually."
        exit 1
    fi
fi

# Print version before
echo "Before upgrade:"
if command -v ads &> /dev/null; then
    ads version
else
    echo "ads not installed globally yet."
fi

echo ""
echo "Building RPM package..."
make rpm

# Find the latest built RPM
RPM_FILE=$(ls -t build/rpm/RPMS/x86_64/ads-*.rpm 2>/dev/null | head -n 1)

if [ -z "$RPM_FILE" ]; then
    echo "Error: RPM file not found in build/rpm/RPMS/x86_64/"
    exit 1
fi

echo ""
echo "Installing/Upgrading RPM: $RPM_FILE"
# Use dnf if available, else fallback to rpm
if command -v dnf &> /dev/null; then
    sudo dnf install -y "$RPM_FILE" || sudo dnf upgrade -y "$RPM_FILE"
else
    sudo rpm -Uvh --force "$RPM_FILE"
fi

echo ""
echo "After upgrade:"
ads version

echo ""
echo "Restarting Konsole tab..."
# If running inside Konsole, replace the current script process with a new shell
if [ -n "$KONSOLE_VERSION" ]; then
    echo "Restarting shell session inside Konsole..."
    exec ${SHELL:-/bin/bash} -l
else
    echo "Not running in Konsole or KONSOLE_VERSION is not set."
    echo "Please restart your terminal manually to ensure the new version is loaded."
    exec ${SHELL:-/bin/bash} -l
fi
