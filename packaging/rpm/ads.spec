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

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}/usr/bin
install -m 755 bin/ads %{buildroot}/usr/bin/ads
install -m 755 bin/ads-shell %{buildroot}/usr/bin/ads-shell
install -m 755 bin/ads-plugin-search %{buildroot}/usr/bin/ads-plugin-search
install -m 755 bin/ads-plugin-llm %{buildroot}/usr/bin/ads-plugin-llm

%clean
rm -rf %{buildroot}

%files
%defattr(-,root,root,-)
/usr/bin/ads
/usr/bin/ads-shell
/usr/bin/ads-plugin-search
/usr/bin/ads-plugin-llm
