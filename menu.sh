#!/bin/bash

set -uo pipefail

# ===============================================
# Konfigurasi Global
# ===============================================
readonly GH_RAW_URL="https://raw.githubusercontent.com"
readonly GH_USERNAME="xenvoid404"
readonly GH_REPO_NAME="neko-tunneling"
readonly GH_BRANCH_NAME="master"
readonly GH_RAW="${GH_RAW_URL}/${GH_USERNAME}/${GH_REPO_NAME}/${GH_BRANCH_NAME}"

readonly CACHE_DIR="/var/lib/nekotun/cache"
readonly CACHE_IP="${CACHE_DIR}/ip"
readonly CACHE_CITY="${CACHE_DIR}/city"
readonly CACHE_ISP="${CACHE_DIR}/isp"
readonly CACHE_DOMAIN="${CACHE_DIR}/domain"

# ===============================================
# Utilitas Output
# ===============================================
if [[ -t 1 ]] && tput setaf 1 &>/dev/null; then
	NC=$(tput sgr0)
	RED=$(tput setaf 196)
	GREEN=$(tput setaf 46)
	BLUE1=$(tput setaf 33)
	BLUE2=$(tput setaf 27)
	YELLOW=$(tput setaf 226)
	ORANGE1=$(tput setaf 214)
	WHITE=$(tput setaf 15)
	HEADER="$(tput setaf 15)$(tput setab 129)"
else
	NC="" RED="" GREEN="" BLUE1="" BLUE2="" YELLOW="" ORANGE1="" WHITE="" HEADER=""
fi

readonly NC RED GREEN BLUE1 BLUE2 YELLOW ORANGE1 WHITE HEADER
readonly TL="${BLUE1}┌"
readonly TR="┐${NC}"
readonly BL="${BLUE1}└"
readonly BR="┘${NC}"
readonly VL="${BLUE1}│${NC}"
readonly HL="─────────────────────────────────────────────────"
readonly BULLET="${ORANGE1}⋈${NC}"

# ===============================================
# Utilitas Sistem
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

status_service() {
	if systemctl is-active --quiet "$1"; then
		echo -e "${GREEN}ON${NC}"
	else
		echo -e "${RED}ERR${NC}"
	fi
}

# ===============================================
# Menu SSH
# ===============================================
ssh_menu() {
	while true; do
		clear
		echo -e ""
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${HEADER}                     MENU SSH                    ${NC}"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${BLUE2}  1.)${NC} ${ORANGE1}${BULLET}${NC} Create"
		echo -e "${VL}${BLUE2}  2.)${NC} ${ORANGE1}${BULLET}${NC} Trial"
		echo -e "${VL}${BLUE2}  3.)${NC} ${ORANGE1}${BULLET}${NC} Delete"
		echo -e "${VL}${BLUE2}  4.)${NC} ${ORANGE1}${BULLET}${NC} Renew"
		echo -e "${VL}${BLUE2}  5.)${NC} ${ORANGE1}${BULLET}${NC} Edit Password"
		echo -e "${VL}${BLUE2}  6.)${NC} ${ORANGE1}${BULLET}${NC} List Users"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2}  7.)${NC} ${ORANGE1}${BULLET}${NC} Lock"
		echo -e "${VL}${BLUE2}  8.)${NC} ${ORANGE1}${BULLET}${NC} Unlock"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2}  9.)${NC} ${ORANGE1}${BULLET}${NC} Cek Config"
		echo -e "${VL}${BLUE2} 10.)${NC} ${ORANGE1}${BULLET}${NC} Recovery"
		echo -e "${VL}${BLUE2} 11.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit IP"
		echo -e "${VL}${BLUE2} 12.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit IP All"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2} 13.)${NC} ${ORANGE1}${BULLET}${NC} Back"
		echo -e "${VL}${BLUE2}  x.)${NC} ${ORANGE1}${BULLET}${NC} exit"
		echo -e "${BL}${HL}${BR}"
		echo -e ""

		read -r -p " Pilih Menu [1-13 atau x] : " ssh_menu_opt
		case "$ssh_menu_opt" in
		13)
			break
			;;
		x | X)
			echo -e ""
			echo -e ""
			exit 0
			;;
		*)
			echo -e ""
			echo -e "${RED} Input tidak valid!${NC}"
			echo -e ""
			sleep 1
			continue
			;;
		esac
	done
}

# ===============================================
# Menu Vmess
# ===============================================
vmess_menu() {
	while true; do
		clear
		echo -e ""
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${HEADER}                    MENU VMESS                   ${NC}"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${BLUE2}  1.)${NC} ${ORANGE1}${BULLET}${NC} Create"
		echo -e "${VL}${BLUE2}  2.)${NC} ${ORANGE1}${BULLET}${NC} Trial"
		echo -e "${VL}${BLUE2}  3.)${NC} ${ORANGE1}${BULLET}${NC} Delete"
		echo -e "${VL}${BLUE2}  4.)${NC} ${ORANGE1}${BULLET}${NC} Renew"
		echo -e "${VL}${BLUE2}  5.)${NC} ${ORANGE1}${BULLET}${NC} Edit UUID"
		echo -e "${VL}${BLUE2}  6.)${NC} ${ORANGE1}${BULLET}${NC} List Users"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2}  7.)${NC} ${ORANGE1}${BULLET}${NC} Lock"
		echo -e "${VL}${BLUE2}  8.)${NC} ${ORANGE1}${BULLET}${NC} Unlock"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2}  9.)${NC} ${ORANGE1}${BULLET}${NC} Cek Config"
		echo -e "${VL}${BLUE2} 10.)${NC} ${ORANGE1}${BULLET}${NC} Recovery"
		echo -e "${VL}${BLUE2} 11.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit IP"
		echo -e "${VL}${BLUE2} 12.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit IP All"
		echo -e "${VL}${BLUE2} 13.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit Quota"
		echo -e "${VL}${BLUE2} 14.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit Quota All"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2} 15.)${NC} ${ORANGE1}${BULLET}${NC} Back"
		echo -e "${VL}${BLUE2}  x.)${NC} ${ORANGE1}${BULLET}${NC} exit"
		echo -e "${BL}${HL}${BR}"
		echo -e ""

		read -r -p " Pilih Menu [1-15 atau x] : " vmess_menu_opt
		case "$vmess_menu_opt" in
		15)
			break
			;;
		x | X)
			echo -e ""
			echo -e ""
			exit 0
			;;
		*)
			echo -e ""
			echo -e "${RED} Input tidak valid!${NC}"
			echo -e ""
			sleep 1
			continue
			;;
		esac
	done
}

# ===============================================
# Menu Vless
# ===============================================
vless_menu() {
	while true; do
		clear
		echo -e ""
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${HEADER}                    MENU VLESS                   ${NC}"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${BLUE2}  1.)${NC} ${ORANGE1}${BULLET}${NC} Create"
		echo -e "${VL}${BLUE2}  2.)${NC} ${ORANGE1}${BULLET}${NC} Trial"
		echo -e "${VL}${BLUE2}  3.)${NC} ${ORANGE1}${BULLET}${NC} Delete"
		echo -e "${VL}${BLUE2}  4.)${NC} ${ORANGE1}${BULLET}${NC} Renew"
		echo -e "${VL}${BLUE2}  5.)${NC} ${ORANGE1}${BULLET}${NC} Edit UUID"
		echo -e "${VL}${BLUE2}  6.)${NC} ${ORANGE1}${BULLET}${NC} List Users"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2}  7.)${NC} ${ORANGE1}${BULLET}${NC} Lock"
		echo -e "${VL}${BLUE2}  8.)${NC} ${ORANGE1}${BULLET}${NC} Unlock"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2}  9.)${NC} ${ORANGE1}${BULLET}${NC} Cek Config"
		echo -e "${VL}${BLUE2} 10.)${NC} ${ORANGE1}${BULLET}${NC} Recovery"
		echo -e "${VL}${BLUE2} 11.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit IP"
		echo -e "${VL}${BLUE2} 12.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit IP All"
		echo -e "${VL}${BLUE2} 13.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit Quota"
		echo -e "${VL}${BLUE2} 14.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit Quota All"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2} 15.)${NC} ${ORANGE1}${BULLET}${NC} Back"
		echo -e "${VL}${BLUE2}  x.)${NC} ${ORANGE1}${BULLET}${NC} exit"
		echo -e "${BL}${HL}${BR}"
		echo -e ""

		read -r -p " Pilih Menu [1-15 atau x] : " vless_menu_opt
		case "$vless_menu_opt" in
		15)
			break
			;;
		x | X)
			echo -e ""
			echo -e ""
			exit 0
			;;
		*)
			echo -e ""
			echo -e "${RED} Input tidak valid!${NC}"
			echo -e ""
			sleep 1
			continue
			;;
		esac
	done
}

# ===============================================
# Menu Trojan
# ===============================================
trojan_menu() {
	while true; do
		clear
		echo -e ""
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${HEADER}                   MENU TROJAN                   ${NC}"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${BLUE2}  1.)${NC} ${ORANGE1}${BULLET}${NC} Create"
		echo -e "${VL}${BLUE2}  2.)${NC} ${ORANGE1}${BULLET}${NC} Trial"
		echo -e "${VL}${BLUE2}  3.)${NC} ${ORANGE1}${BULLET}${NC} Delete"
		echo -e "${VL}${BLUE2}  4.)${NC} ${ORANGE1}${BULLET}${NC} Renew"
		echo -e "${VL}${BLUE2}  5.)${NC} ${ORANGE1}${BULLET}${NC} Edit UUID"
		echo -e "${VL}${BLUE2}  6.)${NC} ${ORANGE1}${BULLET}${NC} List Users"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2}  7.)${NC} ${ORANGE1}${BULLET}${NC} Lock"
		echo -e "${VL}${BLUE2}  8.)${NC} ${ORANGE1}${BULLET}${NC} Unlock"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2}  9.)${NC} ${ORANGE1}${BULLET}${NC} Cek Config"
		echo -e "${VL}${BLUE2} 10.)${NC} ${ORANGE1}${BULLET}${NC} Recovery"
		echo -e "${VL}${BLUE2} 11.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit IP"
		echo -e "${VL}${BLUE2} 12.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit IP All"
		echo -e "${VL}${BLUE2} 13.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit Quota"
		echo -e "${VL}${BLUE2} 14.)${NC} ${ORANGE1}${BULLET}${NC} Edit Limit Quota All"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2} 15.)${NC} ${ORANGE1}${BULLET}${NC} Back"
		echo -e "${VL}${BLUE2}  x.)${NC} ${ORANGE1}${BULLET}${NC} exit"
		echo -e "${BL}${HL}${BR}"
		echo -e ""

		read -r -p " Pilih Menu [1-15 atau x] : " trojan_menu_opt
		case "$trojan_menu_opt" in
		15) break ;;
		x | X)
			echo -e ""
			echo -e ""
			exit 0
			;;
		*)
			echo -e ""
			echo -e "${RED} Input tidak valid!${NC}"
			echo -e ""
			sleep 1
			continue
			;;
		esac
	done
}

# ===============================================
# Menu Features
# ===============================================
features_check_bandwidth() {
	while true; do
		clear
		echo -e ""
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${HEADER}                  CEK BANDWIDTH                  ${NC}"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${BLUE2} 1.)${NC} ${ORANGE1}${BULLET}${NC} Cek Bandwidth Bulanan"
		echo -e "${VL}${BLUE2} 2.)${NC} ${ORANGE1}${BULLET}${NC} Cek Bandwidth Harian"
		echo -e "${VL}${BLUE2} 3.)${NC} ${ORANGE1}${BULLET}${NC} Cek Bandwidth 5 Menit"
		echo -e "${VL}${BLUE2} 4.)${NC} ${ORANGE1}${BULLET}${NC} Cek Bandwidth Realtime"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2} 5.)${NC} ${ORANGE1}${BULLET}${NC} Back"
		echo -e "${VL}${BLUE2} x.)${NC} ${ORANGE1}${BULLET}${NC} exit"
		echo -e "${BL}${HL}${BR}"
		echo -e ""

		read -r -p " Pilih Menu [1-5 atau x] : " features_check_bandwidth_opt
		case "$features_check_bandwidth_opt" in
		1)
			clear
			echo -e "${TL}${HL}${TR}"
			echo -e "${VL}${HEADER}                 BANDWIDTH BULANAN               ${NC}"
			echo -e "${BL}${HL}${BR}"
			vnstat -m
			echo ""
			echo -e "${WHITE} Tekan [Enter] untuk kembali...${NC}"
			read -r
			;;
		2)
			clear
			echo -e "${TL}${HL}${TR}"
			echo -e "${VL}${HEADER}                 BANDWIDTH HARIAN                ${NC}"
			echo -e "${BL}${HL}${BR}"
			vnstat -d
			echo ""
			echo -e "${WHITE} Tekan [Enter] untuk kembali...${NC}"
			read -r
			;;
		3)
			clear
			echo -e "${TL}${HL}${TR}"
			echo -e "${VL}${HEADER}                BANDWIDTH 5 MENIT                ${NC}"
			echo -e "${BL}${HL}${BR}"
			vnstat -5
			echo ""
			echo -e "${WHITE} Tekan [Enter] untuk kembali...${NC}"
			read -r
			;;
		4)
			clear
			echo -e "${TL}${HL}${TR}"
			echo -e "${VL}${HEADER}               BANDWIDTH REALTIME                ${NC}"
			echo -e "${BL}${HL}${BR}"
			echo ""
			vnstat -tr
			echo ""
			echo -e "${WHITE} Tekan [Enter] untuk kembali...${NC}"
			read -r
			;;
		5) break ;;
		x | X)
			echo -e ""
			echo -e ""
			exit 0
			;;
		*)
			echo -e ""
			echo -e "${RED} Input tidak valid!${NC}"
			echo -e ""
			sleep 1
			continue
			;;
		esac
	done
}

features_menu() {
	while true; do
		clear
		echo -e ""
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${HEADER}                  MENU FEATURES                  ${NC}"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${BLUE2} 1.)${NC} ${ORANGE1}${BULLET}${NC} Cek Bandwidth"
		echo -e "${VL}${BLUE2} 2.)${NC} ${ORANGE1}${BULLET}${NC} Auto Reboot"
		echo -e "${VL}${BLUE2} 3.)${NC} ${ORANGE1}${BULLET}${NC} Ubah Versi Dropbear"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2} 4.)${NC} ${ORANGE1}${BULLET}${NC} Backup"
		echo -e "${VL}${BLUE2} 5.)${NC} ${ORANGE1}${BULLET}${NC} Restore"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2} 6.)${NC} ${ORANGE1}${BULLET}${NC} Ubah Domain"
		echo -e "${VL}${BLUE2} 7.)${NC} ${ORANGE1}${BULLET}${NC} Ubah Banner"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL}${BLUE2} 8.)${NC} ${ORANGE1}${BULLET}${NC} Back"
		echo -e "${VL}${BLUE2} x.)${NC} ${ORANGE1}${BULLET}${NC} exit"
		echo -e "${BL}${HL}${BR}"
		echo -e ""

		read -r -p " Pilih Menu [1-8 atau x] : " features_menu_opt
		case "$features_menu_opt" in
		1) features_check_bandwidth ;;
		8) break ;;
		x | X)
			echo -e ""
			echo -e ""
			exit 0
			;;
		*)
			echo -e ""
			echo -e "${RED} Input tidak valid!${NC}"
			echo -e ""
			sleep 1
			continue
			;;
		esac
	done
}

# ===============================================
# Menu Utama
# ===============================================
get_os_name() {
	if [[ -r /etc/os-release ]]; then
		source /etc/os-release
		echo "${PRETTY_NAME//\"/}"
	else
		echo "Unknown"
	fi
}
readonly OS_NAME="$(get_os_name)"

get_ip_info() {
	local file="$1"
	local field="$2"
	mkdir -p "$CACHE_DIR"
	ensure_pkg curl jq

	if [[ ! -s "$file" ]]; then
		local response
		response=$(curl -fsS --max-time 5 http://ip-api.com/json/) || return 1
		echo "$response" | jq -r ".$field" >"$file"
	fi

	cat "$file" 2>/dev/null || echo "Unknown"
}
readonly IP_INFO="$(get_ip_info "$CACHE_IP" query)"
readonly CITY_INFO="$(get_ip_info "$CACHE_CITY" city)"
readonly ISP_INFO="$(get_ip_info "$CACHE_ISP" isp)"

get_vnstat_info() {
	ensure_pkg vnstat jq

	local iface json month day live
	iface=$(vnstat --json | jq -r '.interfaces[0].name')
	[[ -n "$iface" ]] || return 1

	json=$(vnstat -i "$iface" --json) || return 1
	month=$(jq '.interfaces[0].traffic.month[-1]' <<<"$json")
	day=$(jq '.interfaces[0].traffic.day[-1]' <<<"$json")

	local jq_fmt='
		def human:
			if . == null then "0 B"
			elif . >= 1073741824 then "\((. / 1073741824 * 100 | round) / 100) GiB"
			elif . >= 1048576 then "\((. / 1048576 * 100 | round) / 100) MiB"
			elif . >= 1024 then "\((. / 1024 * 100 | round) / 100) KiB"
			else "\(.) B"
			end;
	'

	MONTH_RX=$(jq -r "${jq_fmt} .rx | human" <<<"$month")
	MONTH_TX=$(jq -r "${jq_fmt} .tx | human" <<<"$month")
	MONTH_TOT=$(jq -r "${jq_fmt} (.rx + .tx) | human" <<<"$month")

	TODAY_RX=$(jq -r "${jq_fmt} .rx | human" <<<"$day")
	TODAY_TX=$(jq -r "${jq_fmt} .tx | human" <<<"$day")
	TODAY_TOT=$(jq -r "${jq_fmt} (.rx + .tx) | human" <<<"$day")

	live=$(vnstat -i "$iface" -tr 2 --json 2>/dev/null)
	if [[ -n "$live" ]]; then
		local rx_rate tx_rate
		rx_rate=$(jq -r '.rx.ratestring // "0 bit/s"' <<<"$live")
		tx_rate=$(jq -r '.tx.ratestring // "0 bit/s"' <<<"$live")
		REALTIME="${ORANGE1}↓${NC} ${rx_rate}  ${ORANGE1}↑${NC} ${tx_rate}"
	else
		REALTIME="N/A"
	fi
}

main_check_services() {
	while true; do
		clear
		echo -e ""
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${HEADER}                  STATUS SERVICE                 ${NC}"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL} OpenSSH           :  $(status_service ssh)"
		echo -e "${VL} Dropbear          :  $(status_service dropbear)"
		echo -e "${VL} Xray              :  $(status_service xray)"
		echo -e "${VL} Badvpn            :  $(status_service badvpn)"
		echo -e "${VL} API               :  $(status_service gokil)"
		echo -e "${VL} Multiplexer       :  $(status_service gosip)"
		echo -e "${BL}${HL}${BR}"
		echo -e ""
		echo -e ""
		echo -e "${WHITE} Tekan [Enter] untuk kembali...${NC}"
		read -r
		break
	done
}

main_menu() {
	while true; do
		clear
		echo -e ""
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${HEADER}                  NEKO TUNNELING                 ${NC}"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL} OS       : ${OS_NAME}"
		echo -e "${VL} RAM      : $(free -m | awk '/Mem:/ {print $3" MB / "$2" MB"}')"
		echo -e "${VL} SWAP     : $(free -m | awk '/Swap:/ {print $3" MB / "$2" MB"}')"
		echo -e "${VL} CITY     : ${CITY_INFO}"
		echo -e "${VL} ISP      : ${ISP_INFO}"
		echo -e "${VL} IP       : ${BLUE1}${IP_INFO}${NC}"
		echo -e "${VL} DOMAIN   : ${BLUE1}$(cat "$CACHE_DOMAIN" 2>/dev/null || echo "Unknown")${NC}                        "
		echo -e "${VL} UPTIME   : $(uptime -p | sed 's/up //')"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		get_vnstat_info
		echo -e "${VL} MONTH    : ${MONTH_TOT}    [$(date +"%B")]"
		echo -e "${VL} RX       : ${MONTH_RX}"
		echo -e "${VL} TX       : ${MONTH_TX}"
		echo -e "${VL}${BLUE1}${HL}${NC}"
		echo -e "${VL} DAY      : ${TODAY_TOT}  [$(date +"%A")]"
		echo -e "${VL} RX       : ${TODAY_RX}"
		echo -e "${VL} TX       : ${TODAY_TX}"
		echo -e "${VL} REALTIME : ${REALTIME}"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}     XRAY : $(status_service xray)   |   SSH : $(status_service dropbear)   |   MUX : $(status_service gosip)"
		echo -e "${BL}${HL}${BR}"
		echo -e "${TL}${HL}${TR}"
		echo -e "${VL}${BLUE2}     1.)${NC} ${ORANGE1}${BULLET}${NC} SSH               ${BLUE2}5.)${NC} ${ORANGE1}${BULLET}${NC} FEATURES"
		echo -e "${VL}${BLUE2}     2.)${NC} ${ORANGE1}${BULLET}${NC} VMESS             ${BLUE2}6.)${NC} ${ORANGE1}${BULLET}${NC} SERVICES"
		echo -e "${VL}${BLUE2}     3.)${NC} ${ORANGE1}${BULLET}${NC} VLESS             ${BLUE2}7.)${NC} ${ORANGE1}${BULLET}${NC} REBOOT"
		echo -e "${VL}${BLUE2}     4.)${NC} ${ORANGE1}${BULLET}${NC} TROJAN            ${BLUE2}x.)${NC} ${ORANGE1}${BULLET}${NC} EXIT"
		echo -e "${BL}${HL}${BR}"
		echo ""

		read -r -p " Pilih Menu [1-7 atau x] : " main_menu_opt
		case "$main_menu_opt" in
		1) ssh_menu ;;
		2) vmess_menu ;;
		3) vless_menu ;;
		4) trojan_menu ;;
		5) features_menu ;;
		6) main_check_services ;;
		7) reboot ;;
		x | X)
			echo -e ""
			echo -e ""
			exit 0
			;;
		*)
			echo -e ""
			echo -e "${RED} Input tidak valid!${NC}"
			echo -e ""
			sleep 1
			continue
			;;
		esac
	done
}

trap 'echo -e "${NC}"; echo; exit 0' INT TERM
main_menu
