# Packages the binaries build.sh already produced, mirroring the Debian package
# and PKGBUILD payloads. Version/Release come from the caller so wails.json stays
# the single source of truth:
#   rpmbuild -bb --define "thrm_version 3.7.0" --define "thrm_release 1" ...
#
# No %%{?dist} tag: the released file name is thrm-<version>-<release>.x86_64.rpm,
# and a prebuilt binary is not rebuilt per Fedora release anyway.

# Prebuilt binaries ship stripped (build.sh passes -s -w) and without sources, so
# there is nothing for the debuginfo extractor to work with.
%global debug_package %{nil}

# Version history lives in the GitHub release, so %%changelog stays empty and rpm
# must not try to read a build timestamp out of it.
%global source_date_epoch_from_changelog 0

Name:           thrm
Version:        %{thrm_version}
Release:        %{thrm_release}
Summary:        Flydigi BS series cooler and multi-brand fan controller

License:        MIT
URL:            https://github.com/TIANLI0/THRM

Source0:        thrm
Source1:        thrm-core
Source2:        thrm.desktop
Source3:        thrm.png
Source4:        99-flydigi-fan.rules
Source5:        LICENSE

# Shared library dependencies are generated automatically from the ELF headers,
# which is both more precise than naming packages and keeps this list honest if
# the build's link set changes. Only what cannot be derived is stated here.
# BlueZ is needed for BS1 over BLE, not for USB models, so it stays a weak dep —
# matching Recommends in the Debian package and optdepends in the PKGBUILD.
Recommends:     bluez

ExclusiveArch:  x86_64

%description
THRM controls Flydigi BS series laptop coolers and several built-in laptop fan
controllers. It provides fan curves, temperature monitoring, RGB lighting and
a system tray interface.

This package contains prebuilt binaries.

%prep
# Sources are prebuilt artifacts, not an archive; nothing to unpack or patch.

%build
# Binaries are produced by build.sh before this spec runs.

%install
install -Dm0755 %{SOURCE0} %{buildroot}%{_bindir}/thrm
install -Dm0755 %{SOURCE1} %{buildroot}%{_bindir}/thrm-core
install -Dm0644 %{SOURCE2} %{buildroot}%{_datadir}/applications/thrm.desktop
install -Dm0644 %{SOURCE3} %{buildroot}%{_datadir}/icons/hicolor/256x256/apps/thrm.png
# /usr/lib/udev, not %{_libdir}: udev rules are arch-independent and always live
# in /usr/lib/udev even on x86_64, where %{_libdir} is /usr/lib64.
install -Dm0644 %{SOURCE4} %{buildroot}%{_prefix}/lib/udev/rules.d/99-flydigi-fan.rules
install -Dm0644 %{SOURCE5} %{buildroot}%{_licensedir}/%{name}/LICENSE

%files
%license %{_licensedir}/%{name}/LICENSE
%{_bindir}/thrm
%{_bindir}/thrm-core
%{_datadir}/applications/thrm.desktop
%{_datadir}/icons/hicolor/256x256/apps/thrm.png
%{_prefix}/lib/udev/rules.d/99-flydigi-fan.rules

%post
if command -v udevadm >/dev/null 2>&1; then
    udevadm control --reload-rules || :
    udevadm trigger --subsystem-match=usb --subsystem-match=hidraw || :
fi
# The hidraw rules fall back to GROUP="input" because uaccess only covers an
# active local seat: Bluetooth HID hotplug and non-seat sessions (SSH, user
# services) get no ACL. Package scripts must not edit user groups, so tell
# the admin what to run.
echo "THRM: for Bluetooth HID access, add your user to the input group:"
echo '  sudo usermod -aG input $USER   # re-login afterwards'

%postun
if [ $1 -eq 0 ] && command -v udevadm >/dev/null 2>&1; then
    udevadm control --reload-rules || :
fi

%changelog
