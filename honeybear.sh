#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════════╗
# ║         🐻🍯  HONEY BEAR HONEY POT  — INSTALLER  🍯🐻           ║
# ║        github.com/mikeflynn/honeybearhoneypot                    ║
# ╚══════════════════════════════════════════════════════════════════╝
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/mikeflynn/honeybearhoneypot/main/honeybear.sh | bash
#   curl -fsSL https://... | bash -s -- --upgrade
#   curl -fsSL https://... | bash -s -- --help
set -euo pipefail

# ── Colors & styles ──────────────────────────────────────────────────
RESET="\033[0m"
BOLD="\033[1m"
DIM="\033[2m"
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
CYAN="\033[36m"
MAGENTA="\033[35m"
BG_BLACK="\033[40m"
HONEY="\033[38;5;214m"   # amber/orange — honey colour
BEAR="\033[38;5;130m"    # brown — bear colour

# ── Constants ─────────────────────────────────────────────────────────
REPO="mikeflynn/honeybearhoneypot"
BINARY_NAME="honeybearhoneypot"
BREW_TAP="mikeflynn/honeybearhoneypot"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/${BINARY_NAME}"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"
GITHUB_RELEASES="https://github.com/${REPO}/releases/download"

# Tracks how the binary ended up on the system (brew|direct|build)
INSTALL_METHOD="direct"
# File written during direct/build installs so --version can report back
VERSION_CACHE="${HOME}/.local/share/${BINARY_NAME}/.version"

# ── Helpers ───────────────────────────────────────────────────────────
print_bear() {
  echo -e "${BEAR}${BOLD}"
  echo '        ʕ•ᴥ•ʔ'
  echo '       /|   |\  '
  echo "      ( |   | ) "
  echo '       \|   |/  '
  echo -e "${RESET}"
}

print_banner() {
  echo -e "${BG_BLACK}${HONEY}${BOLD}"
  echo ' ██╗  ██╗ ██████╗ ███╗   ██╗███████╗██╗   ██╗    ██████╗ ███████╗ █████╗ ██████╗ '
  echo ' ██║  ██║██╔═══██╗████╗  ██║██╔════╝╚██╗ ██╔╝    ██╔══██╗██╔════╝██╔══██╗██╔══██╗'
  echo ' ███████║██║   ██║██╔██╗ ██║█████╗   ╚████╔╝     ██████╔╝█████╗  ███████║██████╔╝'
  echo ' ██╔══██║██║   ██║██║╚██╗██║██╔══╝    ╚██╔╝      ██╔══██╗██╔══╝  ██╔══██║██╔══██╗'
  echo ' ██║  ██║╚██████╔╝██║ ╚████║███████╗   ██║       ██████╔╝███████╗██║  ██║██║  ██║'
  echo ' ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═══╝╚══════╝   ╚═╝       ╚═════╝ ╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝'
  echo -e "${RESET}"
  echo -e "${DIM}${CYAN}        🍯  A whimsical SSH honeypot with a very opinionated bear  🍯${RESET}"
  echo ""
}

log()    { echo -e "${CYAN}${BOLD}[≫]${RESET} $*"; }
ok()     { echo -e "${GREEN}${BOLD}[✓]${RESET} $*"; }
warn()   { echo -e "${YELLOW}${BOLD}[!]${RESET} $*"; }
err()    { echo -e "${RED}${BOLD}[✗]${RESET} $*" >&2; }
die()    { err "$*"; exit 1; }
dim()    { echo -e "${DIM}$*${RESET}"; }

typewrite() {
  local msg="$1"
  local delay="${2:-0.03}"
  echo -ne "${MAGENTA}${BOLD}"
  while IFS= read -r -n1 char; do
    echo -ne "$char"
    sleep "$delay" 2>/dev/null || true
  done <<< "$msg"
  echo -e "${RESET}"
}

spinner() {
  local pid=$1
  local msg="${2:-Working...}"
  local frames=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
  local i=0
  while kill -0 "$pid" 2>/dev/null; do
    echo -ne "\r${HONEY}${frames[$((i % ${#frames[@]}))]}${RESET} ${DIM}${msg}${RESET}  "
    sleep 0.1
    ((i++)) || true
  done
  echo -ne "\r\033[K"
}

need_cmd() {
  command -v "$1" &>/dev/null || die "Required command not found: '$1'. Please install it and retry."
}

# ── Homebrew helpers ──────────────────────────────────────────────────
brew_available() {
  command -v brew &>/dev/null
}

# True if the binary was installed (and is managed) by Homebrew
is_brew_managed() {
  brew_available && brew list --formula "$BINARY_NAME" &>/dev/null
}

install_via_brew() {
  log "Adding Homebrew tap ${BOLD}${BREW_TAP}${RESET}..."
  brew tap "$BREW_TAP"
  log "Installing via Homebrew..."
  brew install "$BINARY_NAME"
  INSTALL_METHOD="brew"
  ok "Installed via Homebrew."
}

upgrade_via_brew() {
  log "Upgrading via Homebrew..."
  # Ensure the tap is current before upgrading
  brew update --quiet
  if brew upgrade "$BINARY_NAME" 2>&1 | grep -q "already installed"; then
    ok "Already on the latest version. Nothing to do."
    exit 0
  fi
  INSTALL_METHOD="brew"
  ok "Upgrade complete."
}

# ── Platform detection ────────────────────────────────────────────────
detect_platform() {
  local os arch

  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux)  os="linux"  ;;
    *)      die "Unsupported OS: $(uname -s). Only Linux and macOS are supported." ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64)  arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)             die "Unsupported architecture: $(uname -m)." ;;
  esac

  echo "${os}_${arch}"
}

# ── GitHub release fetching ───────────────────────────────────────────
get_latest_version() {
  local version
  if command -v curl &>/dev/null; then
    version=$(curl -fsSL "$GITHUB_API" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  elif command -v wget &>/dev/null; then
    version=$(wget -qO- "$GITHUB_API" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  else
    die "Neither curl nor wget found. Install one and retry."
  fi

  [[ -z "$version" ]] && die "Could not fetch latest release version from GitHub. Check your connection."
  echo "$version"
}

release_asset_exists() {
  local url="$1"
  if command -v curl &>/dev/null; then
    curl -fsSL --head "$url" &>/dev/null
  else
    wget -q --spider "$url" &>/dev/null
  fi
}

download_file() {
  local url="$1"
  local dest="$2"
  if command -v curl &>/dev/null; then
    curl -fsSL --progress-bar -o "$dest" "$url"
  else
    wget -q --show-progress -O "$dest" "$url"
  fi
}

# ── Checksum verification ─────────────────────────────────────────────
verify_checksum() {
  local file="$1"
  local checksums_file="$2"
  local filename
  filename=$(basename "$file")

  local expected
  expected=$(grep "$filename" "$checksums_file" | awk '{print $1}')
  [[ -z "$expected" ]] && { warn "No checksum entry found for ${filename} — skipping verification."; return 0; }

  local actual
  if command -v sha256sum &>/dev/null; then
    actual=$(sha256sum "$file" | awk '{print $1}')
  elif command -v shasum &>/dev/null; then
    actual=$(shasum -a 256 "$file" | awk '{print $1}')
  else
    warn "No sha256 utility found — skipping checksum verification."
    return 0
  fi

  if [[ "$actual" != "$expected" ]]; then
    die "Checksum mismatch for ${filename}!\n  expected: ${expected}\n  got:      ${actual}"
  fi
  ok "Checksum verified."
}

# ── Install directory setup ───────────────────────────────────────────
pick_install_dir() {
  if [[ -w "$INSTALL_DIR" ]]; then
    echo "$INSTALL_DIR"
  elif command -v sudo &>/dev/null && sudo -n true 2>/dev/null; then
    echo "$INSTALL_DIR"  # sudo available without prompt
  else
    local user_bin="$HOME/.local/bin"
    warn "${INSTALL_DIR} is not writable."
    echo -e "${YELLOW}${BOLD}[?]${RESET} Install to ${BOLD}${user_bin}${RESET} instead? [Y/n] " >&2
    read -r -n1 choice </dev/tty || choice="y"
    echo >&2
    if [[ "${choice,,}" != "n" ]]; then
      mkdir -p "$user_bin"
      echo "$user_bin"
    else
      echo "$INSTALL_DIR"  # caller will need sudo
    fi
  fi
}

install_binary() {
  local src="$1"
  local dest_dir="$2"
  local dest="${dest_dir}/${BINARY_NAME}"

  if [[ -w "$dest_dir" ]]; then
    install -m 755 "$src" "$dest"
  else
    log "Installing to ${dest_dir} (requires sudo)..."
    sudo install -m 755 "$src" "$dest"
  fi
  ok "Installed → ${BOLD}${dest}${RESET}"
}

# ── Installed version check ───────────────────────────────────────────
get_installed_version() {
  # Binary must exist first
  command -v "$BINARY_NAME" &>/dev/null || { echo ""; return; }

  # Brew-managed: ask brew directly (avoids launching the app)
  if is_brew_managed; then
    brew list --versions "$BINARY_NAME" 2>/dev/null | awk '{print $2}'
    return
  fi

  # Use the binary's own --version flag
  "$BINARY_NAME" --version 2>/dev/null && return

  # Last resort: version cache written at install time
  [[ -f "$VERSION_CACHE" ]] && cat "$VERSION_CACHE" && return

  echo "unknown"
}

save_installed_version() {
  local version="$1"
  mkdir -p "$(dirname "$VERSION_CACHE")"
  echo "${version#v}" > "$VERSION_CACHE"
}

# ── Local build fallback ──────────────────────────────────────────────
build_locally() {
  log "Attempting local build from source..."
  need_cmd go

  local tmp_src
  tmp_src=$(mktemp -d)
  trap "rm -rf '$tmp_src'" EXIT

  log "Cloning repository..."
  if command -v git &>/dev/null; then
    git clone --depth 1 "https://github.com/${REPO}.git" "$tmp_src" 2>&1 | \
      grep -v '^remote:' | grep -v 'Cloning into' || true
  else
    die "git is required for local build. Install git and retry."
  fi

  log "Building ${BINARY_NAME}..."
  (
    cd "$tmp_src"
    CGO_ENABLED=1 go build -ldflags="-s -w" -o "${tmp_src}/${BINARY_NAME}" ./main.go
  )

  local dest_dir
  dest_dir=$(pick_install_dir)
  install_binary "${tmp_src}/${BINARY_NAME}" "$dest_dir"

  local built_version
  built_version=$(cat "${tmp_src}/VERSION" 2>/dev/null || echo "source")
  save_installed_version "$built_version"
  INSTALL_METHOD="build"
}

# ── Core install/upgrade logic ────────────────────────────────────────
do_install() {
  local is_upgrade="${1:-false}"

  local platform
  platform=$(detect_platform)
  local os arch
  os="${platform%%_*}"
  arch="${platform##*_}"

  log "Detected platform: ${BOLD}${os}/${arch}${RESET}"

  # ── Homebrew fast-path (macOS only) ───────────────────────────────
  if [[ "$os" == "darwin" ]]; then
    if [[ "$is_upgrade" == "true" ]] && is_brew_managed; then
      log "Installation is Homebrew-managed — using ${BOLD}brew upgrade${RESET}."
      upgrade_via_brew
      return
    fi

    if [[ "$is_upgrade" == "false" ]] && brew_available; then
      echo -e "${HONEY}${BOLD}[🍺]${RESET} Homebrew detected!"
      echo -e "${YELLOW}${BOLD}[?]${RESET} Install via Homebrew? (recommended on macOS) [Y/n] "
      read -r -n1 brew_choice </dev/tty || brew_choice="y"
      echo
      if [[ "${brew_choice,,}" != "n" ]]; then
        install_via_brew
        return
      fi
      log "Skipping Homebrew — falling back to direct download."
    fi
  fi

  # ── Direct download path ───────────────────────────────────────────
  log "Fetching latest release info..."
  local version
  version=$(get_latest_version)
  ok "Latest release: ${BOLD}${version}${RESET}"

  if [[ "$is_upgrade" == "true" ]]; then
    local installed
    installed=$(get_installed_version)
    if [[ -n "$installed" && "$installed" == "${version#v}" ]]; then
      ok "Already on the latest version (${BOLD}${installed}${RESET}). Nothing to do."
      exit 0
    fi
    [[ -n "$installed" ]] && log "Upgrading ${BOLD}${installed}${RESET} → ${BOLD}${version}${RESET}"
  fi

  local archive="${BINARY_NAME}_${os}_${arch}.tar.gz"
  local download_url="${GITHUB_RELEASES}/${version}/${archive}"
  local checksums_url="${GITHUB_RELEASES}/${version}/checksums.txt"

  log "Checking release asset: ${DIM}${archive}${RESET}"

  if ! release_asset_exists "$download_url"; then
    warn "No pre-built release found for ${BOLD}${os}/${arch}${RESET}."
    echo ""
    echo -e "${YELLOW}${BOLD}[?]${RESET} Would you like to build ${BINARY_NAME} from source? [y/N] "
    read -r -n1 build_choice </dev/tty || build_choice="n"
    echo
    if [[ "${build_choice,,}" == "y" ]]; then
      build_locally
      return
    else
      die "No pre-built binary available and local build declined. Exiting."
    fi
  fi

  local tmp_dir
  tmp_dir=$(mktemp -d)
  trap "rm -rf '$tmp_dir'" EXIT

  local archive_path="${tmp_dir}/${archive}"
  local checksums_path="${tmp_dir}/checksums.txt"

  log "Downloading ${archive}..."
  download_file "$download_url" "$archive_path"
  ok "Download complete."

  log "Downloading checksums..."
  download_file "$checksums_url" "$checksums_path"
  verify_checksum "$archive_path" "$checksums_path"

  log "Extracting archive..."
  tar -xzf "$archive_path" -C "$tmp_dir"

  local binary_path="${tmp_dir}/${BINARY_NAME}"
  [[ -f "$binary_path" ]] || die "Binary not found in archive. Release may be malformed."
  chmod +x "$binary_path"

  local dest_dir
  dest_dir=$(pick_install_dir)
  install_binary "$binary_path" "$dest_dir"
  save_installed_version "$version"

  # Config sample
  if [[ ! -f "${HOME}/.config/${BINARY_NAME}/config.json" ]] && [[ -f "${tmp_dir}/config.sample.json" ]]; then
    mkdir -p "${HOME}/.config/${BINARY_NAME}"
    cp "${tmp_dir}/config.sample.json" "${HOME}/.config/${BINARY_NAME}/config.json.sample"
    dim "  → Sample config: ${HOME}/.config/${BINARY_NAME}/config.json.sample"
  fi
}

# ── Post-install notes ────────────────────────────────────────────────
print_post_install() {
  echo ""
  echo -e "${HONEY}${BOLD}╔══════════════════════════════════════════════════════╗"
  echo -e "║           🐻  The bear is ready to trap hackers!     ║"
  echo -e "╚══════════════════════════════════════════════════════╝${RESET}"
  echo ""
  echo -e "  ${BOLD}Quick start:${RESET}"
  echo -e "    ${CYAN}${BOLD}${BINARY_NAME}${RESET}                     # launch GUI + SSH server (port 1337)"
  echo -e "    ${CYAN}${BOLD}${BINARY_NAME} -no-gui${RESET}             # headless mode"
  echo -e "    ${CYAN}${BOLD}${BINARY_NAME} -ssh-port 2222${RESET}      # custom port"
  echo -e "    ${CYAN}${BOLD}${BINARY_NAME} -config config.json${RESET} # with config file"
  echo ""
  echo -e "  ${BOLD}Upgrade later:${RESET}"
  if [[ "$INSTALL_METHOD" == "brew" ]]; then
    echo -e "    ${CYAN}${BOLD}brew upgrade ${BINARY_NAME}${RESET}"
  else
    echo -e "    ${CYAN}${BOLD}curl -fsSL https://raw.githubusercontent.com/${REPO}/main/honeybear.sh | bash -s -- --upgrade${RESET}"
  fi
  echo ""
  if [[ "$INSTALL_METHOD" == "brew" ]]; then
    dim "  → Managed by Homebrew tap ${BREW_TAP}"
  fi
  echo -e "  ${DIM}Docs → https://honeybear.hydrox.fun${RESET}"
  echo ""
}

# ── Help ──────────────────────────────────────────────────────────────
print_help() {
  echo -e "${BOLD}Usage:${RESET}"
  echo "  honeybear.sh [OPTIONS]"
  echo ""
  echo -e "${BOLD}Options:${RESET}"
  echo "  --upgrade    Upgrade to the latest release"
  echo "  --version    Show installed version"
  echo "  --help       Show this help"
  echo ""
  echo -e "${BOLD}Install via Homebrew (macOS):${RESET}"
  echo "  brew tap ${BREW_TAP}"
  echo "  brew install ${BINARY_NAME}"
  echo ""
  echo -e "${BOLD}Install (pipe to bash — Linux or macOS):${RESET}"
  echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/honeybear.sh | bash"
  echo ""
  echo -e "${BOLD}Upgrade (pipe to bash):${RESET}"
  echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/honeybear.sh | bash -s -- --upgrade"
  echo ""
  echo -e "${DIM}On macOS, if Homebrew is detected, the script will offer to use it automatically.${RESET}"
  echo -e "${DIM}If already brew-managed, --upgrade routes through 'brew upgrade' automatically.${RESET}"
}

# ── Entry point ───────────────────────────────────────────────────────
main() {
  local mode="install"

  for arg in "$@"; do
    case "$arg" in
      --upgrade|-u)  mode="upgrade" ;;
      --version|-v)
        local v; v=$(get_installed_version)
        [[ -z "$v" ]] && die "${BINARY_NAME} is not installed." || echo "${BINARY_NAME} ${v}"
        exit 0 ;;
      --help|-h)     print_help; exit 0 ;;
      *) die "Unknown option: ${arg}. Try --help." ;;
    esac
  done

  clear 2>/dev/null || true
  print_banner
  print_bear

  if [[ "$mode" == "upgrade" ]]; then
    typewrite "[ INITIATING UPGRADE SEQUENCE... ]"
  else
    typewrite "[ DEPLOYING BEAR-BASED CYBER DECEPTION INFRASTRUCTURE... ]"
  fi
  echo ""

  do_install "$([[ "$mode" == "upgrade" ]] && echo true || echo false)"
  print_post_install
}

main "$@"
