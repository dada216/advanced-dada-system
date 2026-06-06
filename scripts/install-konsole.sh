#!/bin/bash

# Define paths
KONSOLE_SHARE_DIR="$HOME/.local/share/konsole"
KONSOLE_CONFIG_DIR="$HOME/.config"
PROFILE_NAME="ADS.profile"
KONSOLE_RC="$KONSOLE_CONFIG_DIR/konsolerc"

# Create directories if they don't exist
mkdir -p "$KONSOLE_SHARE_DIR"
mkdir -p "$KONSOLE_CONFIG_DIR"

# Determine ADS binary path
ADS_BIN="/usr/bin/ads"
if [ ! -x "$ADS_BIN" ]; then
    ADS_BIN="$PWD/bin/ads"
fi

# Create Konsole profile
cat > "$KONSOLE_SHARE_DIR/$PROFILE_NAME" <<EOF
[General]
Name=ADS
Command=$ADS_BIN launch
Icon=utilities-terminal

[Appearance]
ColorScheme=Breeze
Font=Hack,12

[Scrolling]
HistoryMode=2
HistorySize=999999
EOF

echo "Created Konsole profile at $KONSOLE_SHARE_DIR/$PROFILE_NAME"

# Update konsolerc to set ADS as default profile
python3 -c "
import os, configparser
rc_path = '$KONSOLE_RC'
config = configparser.ConfigParser()
config.optionxform = str
if os.path.exists(rc_path):
    # Some KDE config files have lines without a section first, configparser hates that.
    # We will ignore errors or prepend a dummy section if needed, but usually it's fine.
    try:
        config.read(rc_path)
    except configparser.MissingSectionHeaderError:
        with open(rc_path, 'r') as f:
            content = '[Dummy]\n' + f.read()
        config.read_string(content)

if 'Desktop Entry' not in config:
    config['Desktop Entry'] = {}
config['Desktop Entry']['DefaultProfile'] = '$PROFILE_NAME'

with open(rc_path, 'w') as f:
    config.write(f, space_around_delimiters=False)

# Clean up dummy section if it was added
with open(rc_path, 'r') as f:
    lines = f.readlines()
with open(rc_path, 'w') as f:
    for line in lines:
        if line.strip() != '[Dummy]':
            f.write(line)
"

echo "Set ADS as default profile in $KONSOLE_RC"
echo "Konsole integration installed successfully!"
