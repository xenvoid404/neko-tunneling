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
			[[ -s "$dest" ]] && return 0
		fi
		((attempt++))
		sleep 2
	done
	die "Gagal mengunduh sumber daya setelah $max_attempt percobaan."
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

main() {
	#check_arch
	#check_root
	#check_virt
	#check_os
	#check_internet

	#setup_first
	setup_swap
}

main "$@"
