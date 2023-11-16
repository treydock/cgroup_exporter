Name:           cgroup_exporter
Version:        0.9.1
Release:        1%{?dist}
Summary:        The cgroup_exporter produces metrics from cgroups.

License:        Apache License
Source0:        %{name}-%{version}.tar.gz
URL:            https://github.com/treydock/cgroup_exporter

BuildRequires:  go-toolset
Requires:       systemd

%description

The cgroup_exporter produces metrics from cgroups.

This exporter by default listens on port 9306 and all metrics are exposed via the /metrics endpoint.

%global debug_package %{nil}

%prep
%autosetup

%build
go build -v -o %{name}

%install
install -Dpm 0755 %{name} %{buildroot}%{_sbindir}/%{name}
install -Dpm 0644 packaging/rpm/%{name}.service %{buildroot}%{_unitdir}/%{name}.service
install -Dpm 0644 packaging/rpm/%{name}.sysconfig %{buildroot}%{_sysconfdir}/sysconfig/%{name}

%clean
rm -rf %{buildroot}

%pre
%{_sbindir}/useradd -c "cgroup exporter user" -s /bin/false -r -d / cgroup_exporter 2>/dev/null || :

%files
%{_sbindir}/%{name}
%{_unitdir}/%{name}.service
%config(noreplace) %{_sysconfdir}/sysconfig/%{name}

%changelog
* Fri Nov 10 2023 Initial RPM
- 
