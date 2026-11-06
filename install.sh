#!/bin/sh

set -eu
umask 077

fail() { printf '%s\n' "Error: $*" >&2; exit 1; }
repository=${SSH_KEYFLOW_REPOSITORY:-}
version=
install_dir=$HOME/.local/bin
profile=
uninstall=false
while [ "$#" -gt 0 ]; do
    case "$1" in
        --repository|--version|--install-dir|--profile)
            [ "$#" -ge 2 ] || fail "Missing value for $1"
            case "$1" in
                --repository) repository=$2 ;;
                --version) version=$2 ;;
                --install-dir) install_dir=$2 ;;
                --profile) profile=$2 ;;
            esac
            shift 2 ;;
        --uninstall) uninstall=true; shift ;;
        *) fail "Unknown argument: $1" ;;
    esac
done
digest() {
    if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'
    else shasum -a 256 "$1" | awk '{print $1}'; fi
}
case "$install_dir" in /*) ;; *) fail 'Install directory must be absolute.' ;; esac
[ ! -L "$install_dir" ] || fail 'Install directory cannot be a symlink.'
mkdir -p "$install_dir"
install_dir=$(cd "$install_dir" && pwd -P)
binary=$install_dir/ssh-keyflow
receipt=$install_dir/.ssh-keyflow-install
[ ! -L "$binary" ] && [ ! -L "$receipt" ] || fail 'Refusing symlink at installation path.'
mkdir "$install_dir/.ssh-keyflow-lock" 2>/dev/null || fail 'Installer is locked; inspect the lock before retrying.'
stage=
committed=false
binary_replaced=false
cleanup() {
    if [ -n "$stage" ]; then
        if [ "$committed" = false ] && [ "$binary_replaced" = true ]; then
            if [ -f "$stage/previous" ]; then mv -f "$stage/previous" "$binary"
            else rm -f "$binary"; rmdir "$receipt"; fi
        fi
        rm -rf "$stage"
    fi
    rmdir "$install_dir/.ssh-keyflow-lock"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
if [ -e "$binary" ]; then
    [ -f "$receipt/hash" ] && [ "$(digest "$binary")" = "$(cat "$receipt/hash")" ] || fail 'Existing binary is unowned or changed; preserved.'
elif [ -e "$receipt" ]; then
    fail 'Installation receipt exists without binary; inspect before retrying.'
fi
if [ "$uninstall" = true ]; then
    if [ -f "$receipt/hash" ]; then
        if [ -f "$receipt/profile" ]; then
            owned_profile=$(cat "$receipt/profile")
            [ -f "$owned_profile" ] && [ ! -L "$owned_profile" ] || fail 'Owned profile is missing or a symlink; preserved.'
            stage=$(mktemp -d "$install_dir/.ssh-keyflow-stage.XXXXXXXX")
            offsets=$(LC_ALL=C awk 'NR==FNR {needle=needle $0 "\n"; next} {haystack=haystack $0 "\n"} END {first=index(haystack,needle); if(first && !index(substr(haystack,first+length(needle)),needle)) print first-1, length(needle)}' "$receipt/stanza" "$owned_profile")
            [ -n "$offsets" ] || fail 'Owned PATH stanza was changed or duplicated; profile and installation preserved.'
            offset=${offsets% *}
            length=${offsets#* }
            dd if="$owned_profile" of="$stage/matched" bs=1 skip="$offset" count="$length" 2>/dev/null
            cmp -s "$stage/matched" "$receipt/stanza" || fail 'Owned PATH stanza was changed or duplicated; profile and installation preserved.'
            dd if="$owned_profile" of="$stage/profile" bs=1 count="$offset" 2>/dev/null
            dd if="$owned_profile" bs=1 skip="$((offset + length))" 2>/dev/null >> "$stage/profile"
            cat "$stage/profile" > "$owned_profile"
            rm "$receipt/profile" "$receipt/stanza"
        fi
        rm "$binary" "$receipt/hash"
        rmdir "$receipt"
    fi
    printf '%s\n' 'Uninstalled owned executable; SSH keys, state, configuration and remote registrations retained.'
    exit 0
fi
printf '%s\n' "$repository" | grep -Eq '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$' || fail 'Specify --repository OWNER/REPO or SSH_KEYFLOW_REPOSITORY.'
case "$(uname -s)" in
    Linux) system=linux ;;
    Darwin) system=darwin ;;
    *) fail 'Unsupported OS; use install.ps1 on Windows.' ;;
esac
case "$(uname -m)" in
    x86_64|amd64) architecture=amd64 ;;
    arm64|aarch64) architecture=arm64 ;;
    *) fail 'Unsupported CPU; supported: x64 and ARM64.' ;;
esac
release_url=${SSH_KEYFLOW_RELEASE_URL:-https://github.com/$repository/releases/download}
api_url=${SSH_KEYFLOW_API_URL:-https://api.github.com/repos/$repository}
fetch() { curl --fail --silent --show-error --location --proto '=https,http' "$1" > "$2"; }
stage=$(mktemp -d "$install_dir/.ssh-keyflow-stage.XXXXXXXX")
if [ -z "$version" ]; then
    fetch "$api_url/releases/latest" "$stage/latest.json"
    grep -Eq '"prerelease"[[:space:]]*:[[:space:]]*false' "$stage/latest.json" && grep -Eq '"draft"[[:space:]]*:[[:space:]]*false' "$stage/latest.json" || fail 'Latest response is not a stable release.'
    version=$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$stage/latest.json")
    printf '%s\n' "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' || fail 'Latest tag is not stable.'
fi
printf '%s\n' "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$' || fail 'Version must be a release tag such as v1.2.3.'
asset=ssh-keyflow_${version}_${system}_${architecture}.tar.gz
fetch "$release_url/$version/$asset" "$stage/$asset"
fetch "$release_url/$version/SHA256SUMS" "$stage/SHA256SUMS"
expected=$(awk -v asset="$asset" '$2 == asset {print $1}' "$stage/SHA256SUMS")
[ "$(awk -v asset="$asset" '$2 == asset {count++} END {print count+0}' "$stage/SHA256SUMS")" = 1 ] || fail 'Missing or ambiguous SHA256 manifest entry.'
printf '%s\n' "$expected" | grep -Eq '^[a-fA-F0-9]{64}$' || fail 'Missing or ambiguous SHA256 manifest entry.'
[ "$(digest "$stage/$asset")" = "$expected" ] || fail 'SHA256 mismatch.'
[ "$(tar -tzf "$stage/$asset")" = ssh-keyflow ] || fail 'Unexpected archive members.'
case "$(tar -tvzf "$stage/$asset")" in -*) ;; *) fail 'Expected a regular archive member.' ;; esac
tar -xzf "$stage/$asset" -C "$stage" ssh-keyflow
[ -f "$stage/ssh-keyflow" ] && [ ! -L "$stage/ssh-keyflow" ] || fail 'Expected a regular executable.'
chmod 755 "$stage/ssh-keyflow"
digest "$stage/ssh-keyflow" > "$stage/hash"
if [ -f "$binary" ]; then cp -p "$binary" "$stage/previous"; fi
mkdir -p "$receipt"
mv -f "$stage/ssh-keyflow" "$binary"
binary_replaced=true
mv -f "$stage/hash" "$receipt/hash"
committed=true
printf 'Installed %s to %s\n' "$version" "$install_dir/ssh-keyflow"
quoted_dir=$(printf '%s' "$install_dir" | sed "s/'/'\\\\''/g")
printf "Manual PATH command: export PATH='%s':\"\$PATH\"\n" "$quoted_dir"
if [ -z "$profile" ]; then
    case "${SHELL:-}" in
        */bash) if [ "$system" = darwin ]; then profile=$HOME/.bash_profile; else profile=$HOME/.bashrc; fi ;;
        */zsh) profile=$HOME/.zshrc ;;
        */sh) profile=$HOME/.profile ;;
        *) printf '%s\n' 'Unknown shell: add the PATH command manually.'; exit 0 ;;
    esac
fi
case ":$PATH:" in *":$install_dir:"*) exit 0 ;; esac
[ ! -f "$receipt/profile" ] || exit 0
case "$profile" in /*) ;; *) fail 'Profile must be absolute.' ;; esac
case "$profile$install_dir" in *'
'*) fail 'Newlines are not supported in install/profile paths.' ;; esac
[ ! -L "$profile" ] || fail 'Profile cannot be a symlink.'
printf "\n: 'SSH KeyFlow PATH begin'\ncase \":\$PATH:\" in *:'%s':*) ;; *) export PATH='%s':\"\$PATH\" ;; esac\n: 'SSH KeyFlow PATH end'\n" "$quoted_dir" "$quoted_dir" > "$stage/stanza"
printf 'Proposed addition to %s:\n' "$profile"
cat "$stage/stanza"
printf 'Persist this PATH addition? [y/N] '
answer=
read -r answer || true
answer=$(printf '%s' "$answer" | tr -d '\r')
case "$answer" in
    y|Y|yes|YES)
        if [ -f "$profile" ] && grep -F 'SSH KeyFlow PATH begin' "$profile" >/dev/null; then fail 'Unowned PATH marker exists; use manual guidance.'; fi
        mkdir -p "$(dirname "$profile")"
        cp "$stage/stanza" "$receipt/stanza"
        printf '%s\n' "$profile" > "$receipt/profile"
        cat "$stage/stanza" >> "$profile"
        printf '%s\n' 'PATH saved. Open a new shell or source your profile.' ;;
    *) printf '%s\n' 'Profile unchanged. Use the manual PATH command when needed.' ;;
esac
