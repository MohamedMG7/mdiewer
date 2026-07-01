#!/usr/bin/env sh
set -eu

repo="${MDIEWER_REPO:-MohamedMG7/mdiewer}"
version="${MDIEWER_VERSION:-latest}"
install_dir="${MDIEWER_INSTALL_DIR:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  linux) os="linux" ;;
  darwin) os="darwin" ;;
  *) echo "Unsupported OS: $os" >&2; exit 1 ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
esac

asset="mdiewer-${os}-${arch}.tar.gz"
if [ "$version" = "latest" ]; then
  url="https://github.com/${repo}/releases/latest/download/${asset}"
else
  url="https://github.com/${repo}/releases/download/${version}/${asset}"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$install_dir"
echo "Downloading $url"
curl -fsSL "$url" -o "$tmp/$asset"
tar -xzf "$tmp/$asset" -C "$tmp"

binary="$(find "$tmp" -type f -name mdiewer | head -n 1)"
if [ -z "$binary" ]; then
  echo "mdiewer was not found in $asset" >&2
  exit 1
fi

install -m 0755 "$binary" "$install_dir/mdiewer"
echo "Installed mdiewer to $install_dir/mdiewer"

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    echo
    echo "$install_dir is not on PATH."
    echo "Add this to your shell profile:"
    echo "  export PATH=\"$install_dir:\$PATH\""
    ;;
esac

echo
echo "Run: mdiewer --help"
