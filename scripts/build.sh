#!/bin/sh

# Install the matching prebuilt release inside a Herdr-managed plugin checkout.
# Herdr runs this as the manifest's [[build]] command before registering the
# plugin. The downloaded archive is verified against the release checksums.

set -eu

fatal() {
  echo "Namespace Devbox: $1" >&2
  exit 1
}

fetch() {
  url=$1
  output=$2

  if command -v curl >/dev/null 2>&1; then
    curl --fail --location --silent --show-error --output "$output" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget --quiet --output-document "$output" "$url"
  else
    fatal "curl or wget is required to download the release archive"
  fi
}

plugin_root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
manifest="$plugin_root/herdr-plugin.toml"
version=$(sed -n 's/^version = "\([^"]*\)".*/\1/p' "$manifest")

[ -n "$version" ] || fatal "could not read the plugin version from $manifest"

case "$(uname -s 2>/dev/null || echo unknown)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) fatal "only macOS and Linux are supported" ;;
esac

case "$(uname -m 2>/dev/null || echo unknown)" in
  arm64 | aarch64) arch=arm64 ;;
  x86_64 | amd64) arch=amd64 ;;
  *) fatal "no release is available for architecture $(uname -m)" ;;
esac

command -v tar >/dev/null 2>&1 || fatal "tar is required to extract the release archive"

archive="herdr-namespace_${version}_${os}_${arch}.tar.gz"
release_url="https://github.com/nicolasdular/herdr-namespace-devbox/releases/download/v${version}"
temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/herdr-namespace.XXXXXX")
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM

echo "Namespace Devbox: downloading $version for $os/$arch..." >&2
fetch "$release_url/$archive" "$temp_dir/$archive" \
  || fatal "could not download $archive; ensure release v$version exists"
fetch "$release_url/checksums.txt" "$temp_dir/checksums.txt" \
  || fatal "could not download checksums for release v$version"

expected=$(awk -v archive="$archive" '$2 == archive || $2 == "*" archive { print $1 }' "$temp_dir/checksums.txt")
[ -n "$expected" ] || fatal "release checksum for $archive is missing"

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$temp_dir/$archive" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$temp_dir/$archive" | awk '{ print $1 }')
else
  fatal "sha256sum or shasum is required to verify the release archive"
fi

[ "$actual" = "$expected" ] || fatal "checksum verification failed for $archive"

tar -xzf "$temp_dir/$archive" -C "$temp_dir" \
  || fatal "could not extract $archive"
[ -f "$temp_dir/herdr-namespace" ] \
  || fatal "$archive does not contain the herdr-namespace binary"

mkdir -p "$plugin_root/bin"
cp "$temp_dir/herdr-namespace" "$plugin_root/bin/herdr-namespace"
chmod 0755 "$plugin_root/bin/herdr-namespace"

echo "Namespace Devbox: installed $version." >&2
