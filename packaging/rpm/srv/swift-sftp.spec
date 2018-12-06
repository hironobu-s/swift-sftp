Name:           swift-sftp
Version:        %{_version}
Release:        1%{?dist}
Summary:        An SFTP server for OpenStack Swift Object Storage
Group:          System Environment/Daemons
License:        MIT
URL:            https://github.com/hironobu-s/swift-sftp
Source0:	latest.tar.gz
Source1:	%{name}.conf
Source2:	authorized_keys
Source3:	%{name}.service
BuildRoot: 	%{_tmppath}/%{name}-%{version}-%{release}-root

%description
%{summary}

%build
curl -sL https://github.com/hironobu-s/swift-sftp/archive/latest.tar.gz > ${RPM_SOURCE_DIR}/latest.tar.gz

%install
%{__install} -Dp -m 0755 %{_sourcedir}/swift-sftp %{buildroot}%{_sbindir}/%{name}
%{__install} -Dp -m 0600 %{SOURCE1} %{buildroot}%{_sysconfdir}/%{name}/%{name}.conf
%{__install} -Dp -m 0600 %{SOURCE2} %{buildroot}%{_sysconfdir}/%{name}/authorized_keys
%{__install} -Dp -m 0644 %{SOURCE3} %{buildroot}%{_unitdir}/%{name}.service

%clean
%{__rm} -rf %{buildroot}

%files
%defattr(-,root,root,-)
%{_sbindir}/%{name}
%config(noreplace) %{_sysconfdir}/%{name}/%{name}.conf
%config(noreplace) %{_sysconfdir}/%{name}/authorized_keys
%{_unitdir}/%{name}.service

%post
systemctl daemon-reload

%postun
systemctl daemon-reload

%changelog
