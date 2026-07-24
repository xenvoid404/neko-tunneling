#!/bin/bash

set -euo pipefail

# ===============================================
# Konfigurasi Global
# ===============================================
GH_RAW_URL="https://raw.githubusercontent.com"
GH_USERNAME="xenvoid404"
GH_REPO_NAME="neko-tunneling"
GH_BRANCH_NAME="master"
GH_RAW="${GH_RAW_URL}/${GH_USERNAME}/${GH_REPO_NAME}/${GH_BRANCH_NAME}"

export DEBIAN_FRONTEND=noninteractive

# ===============================================
# Konfigurasi Warna
# ===============================================
NC='\x1b[0;39;49m'
RED='\x1b[0;38;5;196;49m'
GREEN='\x1b[0;38;5;46;49m'
BLUE='\x1b[0;38;5;33;49m'
YELLOW='\x1b[0;38;5;226;49m'

# ===============================================
# Utilitas Logging
# ===============================================
print_ok() { echo -e "${GREEN}[OK]${NC} $1"; }
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_err() { echo -e "${RED}[ERR]${NC} $1"; }
die() {
	print_err "$*"
	exit 1
}

# ===============================================
# Utilitas
# ===============================================
ensure_pkg() {
	local missing=()
	local pkg
	for pkg in "$@"; do
		dpkg -s "$pkg" &>/dev/null || missing+=("$pkg")
	done
	if ((${#missing[@]} > 0)); then
		apt install -yq "${missing[@]}"
	fi
}

download() {
	local url="$1" dest="$2"
	local attempt=1
	local max_attempt=3
	while ((attempt <= max_attempt)); do
		if curl -fL --progress-bar --connect-timeout 15 "$url" -o "$dest"; then
			if [[ -s "$dest" ]]; then
				echo ""
				return 0
			fi
		fi
		((attempt++))
		sleep 2
	done
	die "Gagal mengunduh sumber daya setelah $max_attempt percobaan."
}

disable_service() {
	local svc="$1"
	if systemctl is-active --quiet "$svc" 2>/dev/null; then
		systemctl stop "$svc" 2>/dev/null || true
	fi
	systemctl disable "$svc" 2>/dev/null || true
}

kill_port() {
	for port in "$@"; do
		local pids
		pids=$(ss -tulwnp 2>/dev/null | grep -P ":${port}(?=\s|,|$)" | grep -oP 'pid=\K[0-9]+' | sort -u) || true
		if [[ -n "$pids" ]]; then
			kill -9 $pids 2>/dev/null || true
		fi
	done
}

reset_dir() {
	local dir="$1"
	if [[ -d "$dir" ]]; then
		rm -rf "$dir"
	fi
	mkdir -p "$dir"
}

verify_service() {
	local svc="$1"
	if systemctl is-active --quiet "$svc"; then
		print_ok "Service ${svc} aktif."
		return 0
	fi
	print_err "Service ${svc} gagal jalan. Log terakhir:"
	journalctl -u "$svc" -n 15 --no-pager 2>/dev/null || true
	return 1
}

run_with_timer() {
	local pid=$1
	local msg=$2
	local seconds=0

	echo -ne "${BLUE}[INFO]${NC} ${msg}... 000"

	while kill -0 "$pid" 2>/dev/null; do
		sleep 1
		seconds=$((seconds + 1))
		printf "\b\b\b%03d" "$seconds"
	done

	printf "\b\b\b%03d ${GREEN}Selesai${NC}\n" "$seconds"
}

# ===============================================
# Tahap Pemeriksaan Awal
# ===============================================
check_arch() {
	print_info "Cek arsitektur CPU"
	local arch
	arch=$(uname -m)
	[[ "$arch" == "x86_64" ]] ||
		die "Arsitektur ${arch} tidak didukung. Script ini hanya untuk amd64 (x86_64)."
}

check_root() {
	print_info "Cek hak akses root"
	[[ $EUID -eq 0 ]] ||
		die "Jalankan sebagai root (sudo atau login root langsung)."
}

check_virt() {
	print_info "Cek jenis virtualisasi"
	if command -v systemd-detect-virt &>/dev/null; then
		local virt
		virt=$(systemd-detect-virt)
		[[ "$virt" != "openvz" ]] ||
			die "Terdeteksi OpenVZ — tidak didukung."
	fi
}

check_os() {
	print_info "Cek sistem operasi"
	[[ -f /etc/os-release ]] || die "Tidak bisa mendeteksi OS (/etc/os-release tidak ada)."

	# shellcheck disable=SC1091
	source /etc/os-release
	local major="${VERSION_ID%%.*}"

	case "$ID" in
	ubuntu) ((major >= 22)) || die "Butuh Ubuntu 22.04+, ini ${VERSION_ID}." ;;
	debian) ((major >= 11)) || die "Butuh Debian 11+, ini ${VERSION_ID}." ;;
	*) die "OS tidak didukung: ${ID}" ;;
	esac

	print_ok "${ID} ${VERSION_ID}, amd64 — kompatibel."
}

check_internet() {
	print_info "Cek koneksi internet"
	if ping -c1 -W2 1.1.1.1 &>/dev/null || ping -c1 -W2 8.8.8.8 &>/dev/null; then
		print_ok "Internet tersedia"
		return 0
	fi

	print_warn "Mencoba memperbaiki DNS..."
	echo "nameserver 1.1.1.1" >>/etc/resolv.conf
	echo "nameserver 8.8.8.8" >>/etc/resolv.conf

	ping -c1 -W2 1.1.1.1 &>/dev/null || ping -c1 -W2 8.8.8.8 &>/dev/null || die "Tidak ada koneksi internet. Cek jaringan VPS."
	print_ok "Internet tersedia"
}

# ===============================================
# Tahap Instalasu
# ===============================================
setup_first() {
	print_info "Konfigurasi dasar sistem"

	apt remove --purge -yq apache2 nginx 2>/dev/null || true
	apt autoremove -yq || true
	apt clean -q || true
	apt update -yq
	ensure_pkg curl wget ntp

	download "${GH_RAW}/config/.profile" /root/.profile
	cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak
	download "${GH_RAW}/config/sshd_config" /etc/ssh/sshd_config

	if ! sshd -t; then
		print_err "Konfigurasi sshd tidak valid!"
		print_warn "Melakukan rollback..."
		mv /etc/ssh/sshd_config.bak /etc/ssh/sshd_config
		die "Rollback sshd_config selesai, instalasi dibatalkan."
	fi

	rm -f /etc/ssh/sshd_config.bak
	systemctl restart sshd || true
	print_ok "Setup dasar selesai."
}

setup_swap() {
	print_info "Konfigurasi swap RAM"

	local current_mb target_mb=2048
	current_mb=$(free -m | awk '/^Swap:/ {print $2}')
	current_mb=${current_mb:-0}

	if ((current_mb < target_mb)); then
		local deficit=$((target_mb - current_mb))
		local idx=1

		while ((deficit > 0)); do
			local size=$deficit
			((size > 1024)) && size=1024

			local file="/swapfile${idx}"
			while [[ -f "$file" ]]; do
				((idx++))
				file="/swapfile${idx}"
			done

			dd if=/dev/zero of="$file" bs=1M count="$size" status=progress
			chmod 600 "$file"
			mkswap "$file"
			swapon "$file"

			grep -q "$file" /etc/fstab || echo "$file none swap sw 0 0" >>/etc/fstab
			deficit=$((deficit - size))
			((idx++))
		done
	fi

	local level=60
	sysctl "vm.swappiness=${level}" 2>/dev/null || true

	if grep -q "^vm.swappiness" /etc/sysctl.conf; then
		sed -i "s/^vm.swappiness.*/vm.swappiness=${level}/" /etc/sysctl.conf
	else
		echo "vm.swappiness=${level}" >>/etc/sysctl.conf
	fi

	print_ok "Swap RAM berhasil diatur."
}

setup_dropbear() {
	print_info "Instal dropbear"
	disable_service dropbear
	kill_port 90 143 69

	local need_ver="2019.78"
	local current_ver
	current_ver=$(dropbear -V 2>&1 || true)

	if [[ "$current_ver" == *"$need_ver"* ]]; then
		print_info "Dropbear versi $need_ver sudah terpasang, lewati kompilasi ulang."
	else
		ensure_pkg build-essential zlib1g-dev bzip2
		local build_dir="/tmp/dropbear-build"
		rm -rf "$build_dir"
		mkdir -p "$build_dir"
		pushd "$build_dir" >/dev/null || die "Gagal masuk ${build_dir}"

		download "${GH_RAW}/dropbear/dropbear-${need_ver}.tar.bz2" "dropbear-${need_ver}.tar.bz2"
		tar -xf "dropbear-${need_ver}.tar.bz2"
		cd "dropbear-${need_ver}"

		./configure --prefix=/usr --sbindir=/usr/sbin --bindir=/usr/bin >/dev/null 2>&1 &
		run_with_timer $! "Menyiapkan konfigurasi"
		wait $! || die "Kompilasi gagal di tahap configure!"

		make >/dev/null 2>&1 &
		run_with_timer $! "Mengkompilasi source code"
		wait $! || die "Kompilasi gagal di tahap make!"

		make install >/dev/null 2>&1 &
		run_with_timer $! "Menginstal binary"
		wait $! || die "Instalasi gagal di tahap make install!"

		print_ok "Kompilasi Dropbear selesai"

		popd >/dev/null
		rm -rf "$build_dir"
	fi

	reset_dir /etc/dropbear
	[[ -f /etc/dropbear/dropbear_rsa_host_key ]] || /usr/bin/dropbearkey -t rsa -f /etc/dropbear/dropbear_rsa_host_key
	[[ -f /etc/dropbear/dropbear_ecdsa_host_key ]] || /usr/bin/dropbearkey -t ecdsa -f /etc/dropbear/dropbear_ecdsa_host_key

	grep -q "/bin/false" /etc/shells || echo "/bin/false" >>/etc/shells
	grep -q "/usr/sbin/nologin" /etc/shells || echo "/usr/sbin/nologin" >>/etc/shells

	download "${GH_RAW}/config/banner.why" /etc/banner.why
	download "${GH_RAW}/config/systemd/dropbear.service" /usr/lib/systemd/system/dropbear.service

	systemctl daemon-reload
	systemctl enable dropbear >/dev/null 2>&1
	systemctl restart dropbear || true

	verify_service dropbear || die "Dropbear gagal dijalankan."
}

setup_warp() {
	print_info "Instal WARP (wireproxy)"
	disable_service wireproxy
	kill_port 40000
	ensure_pkg curl
	reset_dir /etc/wireproxy
	pushd /etc/wireproxy >/dev/null || die "Gagal masuk /etc/wireproxy"

	download "${GH_RAW}/bin/wgcf" /usr/local/bin/wgcf
	download "${GH_RAW}/bin/wireproxy" /usr/local/sbin/wireproxy
	chmod +x /usr/local/bin/wgcf /usr/local/sbin/wireproxy

	local warp_ready=true
	if [[ ! -f wgcf-account.toml ]]; then
		local attempt=1 max=5 ok=false
		while ((attempt <= max)); do
			if wgcf register --accept-tos --config wgcf-account.toml; then
				ok=true
				break
			fi
			print_warn "Gagal registrasi WARP (${attempt}/${max}). Retry 3 detik..."
			sleep 3
			((attempt++))
		done
		[[ "$ok" == true ]] || warp_ready=false
	fi

	[[ "$warp_ready" != true ]] ||
		wgcf generate --config wgcf-account.toml --profile wgcf-profile.conf

	download "${GH_RAW}/config/warp.conf" /tmp/warp.conf

	local priv pub ep
	priv=$(grep 'PrivateKey' wgcf-profile.conf | cut -d= -f2- | tr -d ' ') || true
	pub=$(grep 'PublicKey' wgcf-profile.conf | cut -d= -f2- | tr -d ' ') || true
	ep=$(grep 'Endpoint' wgcf-profile.conf | cut -d= -f2- | tr -d ' ') || true

	if [[ -z "$priv" || -z "$pub" || -z "$ep" ]]; then
		priv="ISI_MANUAL_PRIVATE_KEY_DISINI"
		pub="ISI_MANUAL_PUBLIC_KEY_DISINI"
		ep="engage.cloudflareclient.com:2408"
		warp_ready=false
	fi

	sed -e "s|__PRIVATE_KEY__|${priv}|g" -e "s|__PUBLIC_KEY__|${pub}|g" -e "s|__ENDPOINT__|${ep}|g" /tmp/warp.conf >warp.conf
	rm -f /tmp/warp.conf
	popd >/dev/null

	download "${GH_RAW}/config/systemd/wireproxy.service" /etc/systemd/system/wireproxy.service

	systemctl daemon-reload
	systemctl enable wireproxy >/dev/null 2>&1

	if [[ "$warp_ready" == true ]]; then
		systemctl restart wireproxy || true
		if systemctl is-active --quiet wireproxy; then
			print_ok "WARP (wireproxy) aktif."
		else
			print_err "Service wireproxy gagal jalan. Log terakhir:"
			journalctl -u wireproxy -n 15 --no-pager 2>/dev/null || true
		fi
	else
		print_warn "File WARP terpasang, tetapi kredensial kosong."
		print_warn "Isi manual di /etc/wireproxy/warp.conf lalu: systemctl restart wireproxy"
	fi
}

setup_xray() {
	print_info "Instal Xray"
	disable_service xray
	kill_port 62789 1054 1055 1056 1057 1058 1059 1060 1061 1062
	ensure_pkg jq unzip

	download "${GH_RAW}/bin/xray" /usr/local/sbin/xray
	chmod +x /usr/local/sbin/xray

	reset_dir /etc/xray
	reset_dir /var/log/xray
	mkdir -p /etc/xray/conf.d
	mkdir -p /var/log/xray
	touch /var/log/xray/access.log /var/log/xray/error.log

	download "${GH_RAW}/config/xray.zip" /tmp/xray.zip
	unzip -oq /tmp/xray.zip -d /etc/xray/conf.d/
	rm -f /tmp/xray.zip

	download "${GH_RAW}/config/geoip.dat" /usr/local/share/xray/geoip.dat
	download "${GH_RAW}/config/geosite.dat" /usr/local/share/xray/geosite.dat
	download "${GH_RAW}/config/systemd/xray.service" /etc/systemd/system/xray.service

	systemctl daemon-reload
	systemctl enable xray >/dev/null 2>&1
	systemctl restart xray || true
	verify_service xray || die "Xray gagal dijalankan."
}

main() {
	#check_arch
	#check_root
	#check_virt
	#check_os
	#check_internet

	#setup_first
	#setup_swap
	#setup_dropbear
	#setup_warp
	setup_xray
}

main "$@"
