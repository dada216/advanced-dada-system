Name:           ads
Version:        {{VERSION}}
Release:        1%{?dist}
Summary:        Advanced Dada System CLI
License:        MIT
URL:            https://github.com/advanced-dada-system/ads
Source0:        ads-%{version}.tar.gz
BuildRequires:  golang >= 1.20, make
Requires:       tmux, openssh-clients, sqlite

%description
CLI-First, Multiplexer-Driven Terminal Analytics Platform.
The platform discards custom PTY wrappers and centralized daemons in favor of standard tmux pipelines and the native ssh binary.

%prep
%setup -q -c

%build
make build

# Generate shell completions natively from the Cobra CLI
./bin/ads completion bash > ads.bash
./bin/ads completion zsh > ads.zsh

# Create global Konsole Profile
cat > ADS.profile <<EOF
[General]
Name=ADS
Command=/usr/bin/ads launch --shell zsh
Icon=utilities-terminal

[Appearance]
ColorScheme=Breeze
Font=Hack,12

[Scrolling]
HistoryMode=2
HistorySize=999999
EOF

# Create standard KDE Desktop Entry
cat > ads.desktop <<EOF
[Desktop Entry]
Name=Advanced Dada System
Comment=Multiplexer-Driven Terminal Analytics Platform
Exec=konsole --profile ADS
Icon=utilities-terminal
Terminal=false
Type=Application
Categories=System;TerminalEmulator;
EOF

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}/usr/bin
install -m 755 bin/ads %{buildroot}/usr/bin/ads
install -m 755 bin/ads-shell %{buildroot}/usr/bin/ads-shell
install -m 755 bin/ads-plugin-search %{buildroot}/usr/bin/ads-plugin-search
install -m 755 bin/ads-plugin-llm %{buildroot}/usr/bin/ads-plugin-llm

# Install shell completions
mkdir -p %{buildroot}/usr/share/bash-completion/completions
mkdir -p %{buildroot}/usr/share/zsh/site-functions
install -m 644 ads.bash %{buildroot}/usr/share/bash-completion/completions/ads
install -m 644 ads.zsh %{buildroot}/usr/share/zsh/site-functions/_ads

# Install KDE Integrations
mkdir -p %{buildroot}/usr/share/konsole
mkdir -p %{buildroot}/usr/share/applications
install -m 644 ADS.profile %{buildroot}/usr/share/konsole/ADS.profile
install -m 644 ads.desktop %{buildroot}/usr/share/applications/ads.desktop

%clean
rm -rf %{buildroot}

%files
%defattr(-,root,root,-)
/usr/bin/ads
/usr/bin/ads-shell
/usr/bin/ads-plugin-search
/usr/bin/ads-plugin-llm
/usr/share/bash-completion/completions/ads
/usr/share/zsh/site-functions/_ads
/usr/share/konsole/ADS.profile
/usr/share/applications/ads.desktop
