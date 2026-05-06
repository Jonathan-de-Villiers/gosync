#!/usr/bin/env bash
set -euo pipefail

DOTFILES="$HOME/dotfiles"
BACKUP_ROOT="$HOME/dotfiles-backups"
BACKUP_DIR="$BACKUP_ROOT/$(date +%Y-%m-%d_%H-%M-%S)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

show_help() {
  cat <<EOF
Dotfiles sync helper

Usage:
  ./sync-dotfiles.sh pull <package|all>
  ./sync-dotfiles.sh push <package|all>
  ./sync-dotfiles.sh restore <backup-folder-name> <package>
  ./sync-dotfiles.sh backups
  ./sync-dotfiles.sh packages
  ./sync-dotfiles.sh help

Commands:
  pull       Sync from repo to system (Repo -> \$HOME)
  push       Sync from system to repo (\$HOME -> Repo)
  restore    Restore one package from a backup
  backups    List available backups
  packages   List detected packages
  help       Show this help menu

Examples:
  ./sync-dotfiles.sh pull nvim
  ./sync-dotfiles.sh push alacritty
  ./sync-dotfiles.sh push all
  ./sync-dotfiles.sh restore 2026-04-30_14-35-10 nvim

Available packages:
$(get_packages_for_help)
EOF
}

get_packages() {
  find "$DOTFILES" \
    -mindepth 1 \
    -maxdepth 1 \
    -type d \
    ! -name ".git" \
    ! -name ".github" \
    ! -name "scripts" \
    ! -name "backups" \
    ! -name "dotfiles-backups" \
    -printf "%f\n" | sort
}

get_packages_for_help() {
  if [[ -d "$DOTFILES" ]]; then
    get_packages | sed 's/^/  - /'
  else
    echo "  No dotfiles directory found at $DOTFILES"
  fi
}

package_exists() {
  local pkg="$1"
  while read -r existing_pkg; do
    [[ "$existing_pkg" == "$pkg" ]] && return 0
  done < <(get_packages)
  return 1
}

detect_os() {
  case "$(uname -s)" in
    Linux*) echo "linux" ;;
    Darwin*) echo "mac" ;;
    *) echo "unknown" ;;
  esac
}

is_package_allowed_for_os() {
  local pkg="$1"
  local os
  os="$(detect_os)"
  case "$os" in
    linux) return 0 ;;
    mac)
      case "$pkg" in
        hyprland|hyprlauncher|waybar|wofi) return 1 ;;
        *) return 0 ;;
      esac
      ;;
    *) return 0 ;;
  esac
}

get_package_paths() {
  local pkg="$1"
  find "$DOTFILES/$pkg" -type f -printf "%P\n" | sort
}

confirm() {
  local message="$1"
  local answer
  echo -ne "${YELLOW}$message? (y/N): ${NC}"
  read -r answer
  [[ "$answer" == "y" || "$answer" == "Y" ]]
}

get_mod_time() {
  if [[ "$(uname -s)" == "Darwin"* ]]; then
    stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$1" 2>/dev/null || echo "N/A"
  else
    stat -c "%y" "$1" 2>/dev/null | cut -d'.' -f1 || echo "N/A"
  fi
}

show_diffs() {
  local src="$1"
  local dest="$2"
  local rel="$3"

  if [[ -f "$src" && -f "$dest" ]]; then
    if ! diff -q "$src" "$dest" > /dev/null; then
      echo -e "${BLUE}Δ Difference in $rel${NC}"
      echo -e "  Source mod time:      $(get_mod_time "$src")"
      echo -e "  Destination mod time: $(get_mod_time "$dest")"
      echo -e "${YELLOW}--- Diff ---${NC}"
      diff -u --color=always "$src" "$dest" || diff -u "$src" "$dest"
      echo -e "${YELLOW}------------${NC}"
      return 0
    fi
  elif [[ ! -e "$dest" ]]; then
    echo -e "${GREEN}★ New file in source: $rel${NC}"
    return 0
  fi
  return 1
}

sync_path() {
  local pkg="$1"
  local mode="$2"
  local rel="$3"

  local src=""
  local dest=""

  if [[ "$mode" == "pull" ]]; then
    src="$DOTFILES/$pkg/$rel"
    dest="$HOME/$rel"
  else
    src="$HOME/$rel"
    dest="$DOTFILES/$pkg/$rel"
  fi

  if [[ ! -e "$src" ]]; then
    return
  fi

  # Only show and ask if there's a difference
  if show_diffs "$src" "$dest" "$rel"; then
    echo -e "\n${BLUE}==> $mode $pkg: $rel${NC}"
    
    if ! confirm "Apply this sync"; then
      echo "Skipped."
      return
    fi

    mkdir -p "$BACKUP_DIR"
    mkdir -p "$(dirname "$dest")"

    rsync -avh \
      --itemize-changes \
      --backup \
      --backup-dir="$BACKUP_DIR" \
      "$src" "$dest"

    echo -e "${GREEN}Synced: $rel${NC}"
  fi
}

scan_untracked() {
  echo -e "\n${BLUE}🔍 Checking for untracked configs in ~/.config...${NC}"
  for d in "$HOME/.config"/*; do
    [[ -d "$d" ]] || continue
    local pkg_name
    pkg_name=$(basename "$d")

    # Skip large browser and system directories
    [[ "$pkg_name" == "google-chrome" || "$pkg_name" == "Brave-Browser" || "$pkg_name" == "BraveSoftware" || "$pkg_name" == "mozilla" || "$pkg_name" == "systemd" ]] && continue

    # Check if this config is already tracked in any package
    local is_tracked=false
    while read -r pkg; do
      if [[ -d "$DOTFILES/$pkg/.config/$pkg_name" || -f "$DOTFILES/$pkg/.config/$pkg_name" ]]; then
        is_tracked=true
        break
      fi
    done < <(get_packages)

    if [[ "$is_tracked" == false ]]; then
      echo -e "${YELLOW}❓ Untracked config found: ~/.config/$pkg_name${NC}"
      if confirm "Add this to dotfiles?"; then
        read -p "Enter package name for this config (default: $pkg_name): " new_pkg_name
        new_pkg_name=${new_pkg_name:-$pkg_name}
        mkdir -p "$DOTFILES/$new_pkg_name/.config"
        cp -rv "$HOME/.config/$pkg_name" "$DOTFILES/$new_pkg_name/.config/"
        echo -e "${GREEN}✅ Added $pkg_name to package $new_pkg_name${NC}"
      fi
    fi
  done
}

run_package() {
  local mode="$1"
  local pkg="$2"

  if ! package_exists "$pkg"; then
    echo -e "${RED}Unknown package: $pkg${NC}"
    echo ""
    echo "Available packages:"
    get_packages | sed 's/^/  - /'
    exit 1
  fi

  if ! is_package_allowed_for_os "$pkg"; then
    echo -e "${YELLOW}Skipping '$pkg' on $(detect_os).${NC}"
    return
  fi

  while read -r rel; do
    sync_path "$pkg" "$mode" "$rel"
  done < <(get_package_paths "$pkg")
}

list_backups() {
  echo "Available backups:"
  if [[ ! -d "$BACKUP_ROOT" ]]; then
    echo "No backups found."
    return
  fi
  find "$BACKUP_ROOT" \
    -mindepth 1 \
    -maxdepth 1 \
    -type d \
    -printf "%f\n" | sort -r
}

restore_backup() {
  local backup="${1:-}"
  local pkg="${2:-}"

  if [[ -z "$backup" || -z "$pkg" ]]; then
    echo "Usage: ./sync-dotfiles.sh restore <backup-folder-name> <package>"
    echo ""
    list_backups
    exit 1
  fi

  if ! package_exists "$pkg"; then
    echo -e "${RED}Unknown package: $pkg${NC}"
    exit 1
  fi

  local backup_path="$BACKUP_ROOT/$backup"
  if [[ ! -d "$backup_path" ]]; then
    echo -e "${RED}Backup not found: $backup_path${NC}"
    exit 1
  fi

  echo -e "${BLUE}Restoring package: $pkg from $backup_path${NC}"
  local found_any=false
  while read -r rel; do
    if [[ -e "$backup_path/$rel" ]]; then
      found_any=true
      echo -e "\n${YELLOW}==> Restore $rel${NC}"
      rsync -avh --dry-run "$backup_path/$rel" "$HOME/$rel"
    fi
  done < <(get_package_paths "$pkg")

  if [[ "$found_any" != true ]]; then
    echo -e "${RED}No backup files found for package '$pkg'.${NC}"
    exit 1
  fi

  if confirm "Apply restore for $pkg"; then
    while read -r rel; do
      if [[ -e "$backup_path/$rel" ]]; then
        mkdir -p "$(dirname "$HOME/$rel")"
        rsync -avh "$backup_path/$rel" "$HOME/$rel"
      fi
    done < <(get_package_paths "$pkg")
    echo -e "${GREEN}Restore complete: $pkg${NC}"
  fi
}

MODE="${1:-}"
PACKAGE="${2:-}"

case "$MODE" in
  help|-h|--help|"") show_help; exit 0 ;;
  packages) echo "Detected packages:"; get_packages | sed 's/^/  - /'; exit 0 ;;
  backups) list_backups; exit 0 ;;
  restore) restore_backup "${2:-}" "${3:-}"; exit 0 ;;
esac

if [[ "$MODE" != "pull" && "$MODE" != "push" ]]; then
  echo -e "${RED}Invalid command: $MODE${NC}"; show_help; exit 1
fi

if [[ -z "$PACKAGE" ]]; then
  echo -e "${RED}Please provide a package name or 'all'.${NC}"; show_help; exit 1
fi

if [[ "$PACKAGE" == "all" ]]; then
  while read -r pkg; do
    run_package "$MODE" "$pkg"
  done < <(get_packages)
  
  # Scan for untracked configs only during "push all" (syncing system to repo)
  if [[ "$MODE" == "push" ]]; then
    scan_untracked
  fi
else
  run_package "$MODE" "$PACKAGE"
fi

echo -e "\n${GREEN}Done.${NC}"
if [[ -d "$BACKUP_DIR" ]]; then
  echo "Backups saved to: $BACKUP_DIR"
fi
