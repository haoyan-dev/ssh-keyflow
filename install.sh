#!/bin/sh

set -eu
umask 077

fail() { printf '%s\n' "Error: $*" >&2; exit 1; }
repository=${SSH_KEYFLOW_REPOSITORY:-haoyan-dev/ssh-keyflow}
version=
install_dir=$HOME/.local/bin
profile=
uninstall=false
add_to_path=false
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
        --add-to-path) add_to_path=true; shift ;;
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
notice=$install_dir/THIRD_PARTY_NOTICES.txt
receipt=$install_dir/.ssh-keyflow-install
for owned_path in "$binary" "$notice" "$receipt" "$receipt/hash" "$receipt/notices-hash" "$receipt/profile" "$receipt/stanza"; do
    [ ! -L "$owned_path" ] || fail 'Refusing symlink at installation path.'
done
mkdir "$install_dir/.ssh-keyflow-lock" 2>/dev/null || fail 'Installer is locked; inspect the lock before retrying.'
stage=
committed=false
files_replaced=false
receipt_created=false
profile_stage=
profile_replaced=false
profile_target=
backup_installation() {
    if [ -f "$binary" ]; then cp -p "$binary" "$stage/previous"; fi
    if [ -f "$notice" ]; then cp -p "$notice" "$stage/previous-notices"; fi
    for receipt_name in hash notices-hash profile stanza; do
        if [ -f "$receipt/$receipt_name" ]; then cp -p "$receipt/$receipt_name" "$stage/previous-$receipt_name"; fi
    done
}
prepare_profile() {
    profile_target=$1
    [ ! -e "$profile_target" ] || [ -f "$profile_target" ] || fail 'Profile must be a regular file.'
    profile_stage=$(mktemp -d "$(dirname "$profile_target")/.ssh-keyflow-profile.XXXXXXXX")
    if [ -f "$profile_target" ]; then
        cp -p "$profile_target" "$profile_stage/previous"
        cp -p "$profile_target" "$profile_stage/replacement"
    else
        : > "$profile_stage/replacement"
    fi
}
cleanup() {
    if [ "$committed" = false ] && [ "$profile_replaced" = true ]; then
        if [ -f "$profile_stage/previous" ]; then mv -f "$profile_stage/previous" "$profile_target"
        else rm -f "$profile_target"; fi
    fi
    if [ -n "$stage" ]; then
        if [ "$committed" = false ] && [ "$files_replaced" = true ]; then
            if [ -f "$stage/previous" ]; then mv -f "$stage/previous" "$binary"
            else rm -f "$binary"; fi
            if [ -f "$stage/previous-notices" ]; then mv -f "$stage/previous-notices" "$notice"
            else rm -f "$notice"; fi
            mkdir -p "$receipt"
            for receipt_name in hash notices-hash profile stanza; do
                if [ -f "$stage/previous-$receipt_name" ]; then
                    cp -p "$stage/previous-$receipt_name" "$receipt/$receipt_name"
                else rm -f "$receipt/$receipt_name"; fi
            done
            if [ "$receipt_created" = true ]; then rmdir "$receipt"; fi
        fi
        rm -rf "$stage"
    fi
    if [ -n "$profile_stage" ]; then rm -rf "$profile_stage"; fi
    rmdir "$install_dir/.ssh-keyflow-lock"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
if [ -e "$binary" ]; then
    [ -f "$receipt/hash" ] && [ "$(digest "$binary")" = "$(cat "$receipt/hash")" ] || fail 'Existing binary is unowned or changed; preserved.'
elif [ -e "$receipt" ]; then
    fail 'Installation receipt exists without binary; inspect before retrying.'
fi
if [ -e "$notice" ]; then
    [ -f "$notice" ] && [ -f "$receipt/notices-hash" ] && [ "$(digest "$notice")" = "$(cat "$receipt/notices-hash")" ] || fail 'Existing notices are unowned or changed; preserved.'
elif [ -e "$receipt/notices-hash" ]; then
    fail 'Owned notices are missing; inspect before retrying.'
fi
if [ "$uninstall" = true ]; then
    if [ -f "$receipt/hash" ]; then
        stage=$(mktemp -d "$install_dir/.ssh-keyflow-stage.XXXXXXXX")
        backup_installation
        if [ -f "$receipt/profile" ]; then
            owned_profile=$(cat "$receipt/profile")
            [ -f "$owned_profile" ] && [ ! -L "$owned_profile" ] || fail 'Owned profile is missing or a symlink; preserved.'
            offsets=$(LC_ALL=C awk -v BINMODE=3 'NR==FNR {needle=needle $0 "\n"; next} {haystack=haystack $0 "\n"} END {first=index(haystack,needle); if(first && !index(substr(haystack,first+length(needle)),needle)) print first-1, length(needle)}' "$receipt/stanza" "$owned_profile")
            [ -n "$offsets" ] || fail 'Owned PATH stanza was changed or duplicated; profile and installation preserved.'
            offset=${offsets% *}
            length=${offsets#* }
            dd if="$owned_profile" of="$stage/matched" bs=1 skip="$offset" count="$length" 2>/dev/null
            cmp -s "$stage/matched" "$receipt/stanza" || fail 'Owned PATH stanza was changed or duplicated; profile and installation preserved.'
            prepare_profile "$owned_profile"
            dd if="$owned_profile" of="$profile_stage/replacement" bs=1 count="$offset" 2>/dev/null
            dd if="$owned_profile" bs=1 skip="$((offset + length))" conv=notrunc 2>/dev/null >> "$profile_stage/replacement"
            files_replaced=true
            profile_replaced=true
            mv -f "$profile_stage/replacement" "$owned_profile"
            rm "$receipt/profile" "$receipt/stanza"
        fi
        files_replaced=true
        if [ -f "$receipt/notices-hash" ]; then rm "$notice" "$receipt/notices-hash"; fi
        rm "$binary" "$receipt/hash"
        rmdir "$receipt"
    fi
    committed=true
    printf '%s\n' 'Uninstalled owned executable and notices; SSH keys, state, configuration and remote registrations retained.'
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
[ "$(tar -tzf "$stage/$asset" | LC_ALL=C sort)" = "$(printf 'THIRD_PARTY_NOTICES.txt\nssh-keyflow')" ] || fail 'Unexpected archive members.'
tar -tvzf "$stage/$asset" > "$stage/archive-types"
[ "$(wc -l < "$stage/archive-types" | tr -d ' ')" = 2 ] && ! grep -v '^-' "$stage/archive-types" >/dev/null || fail 'Expected regular archive members.'
tar -xzf "$stage/$asset" -C "$stage" ssh-keyflow THIRD_PARTY_NOTICES.txt
[ -f "$stage/ssh-keyflow" ] && [ ! -L "$stage/ssh-keyflow" ] || fail 'Expected a regular executable.'
[ -f "$stage/THIRD_PARTY_NOTICES.txt" ] && [ ! -L "$stage/THIRD_PARTY_NOTICES.txt" ] || fail 'Expected regular notices.'
chmod 755 "$stage/ssh-keyflow"
chmod 644 "$stage/THIRD_PARTY_NOTICES.txt"
digest "$stage/ssh-keyflow" > "$stage/hash"
digest "$stage/THIRD_PARTY_NOTICES.txt" > "$stage/notices-hash"
backup_installation
if [ ! -d "$receipt" ]; then mkdir "$receipt"; receipt_created=true; fi
files_replaced=true
mv -f "$stage/ssh-keyflow" "$binary"
mv -f "$stage/THIRD_PARTY_NOTICES.txt" "$notice"
mv -f "$stage/hash" "$receipt/hash"
mv -f "$stage/notices-hash" "$receipt/notices-hash"
quoted_dir=$(printf '%s' "$install_dir" | sed "s/'/'\\\\''/g")
finish_install() {
    committed=true
    printf 'Installed %s to %s\n' "$version" "$binary"
    printf '%s\n' "Run: $binary"
    printf "Manual PATH command: export PATH='%s':\"\$PATH\"\n" "$quoted_dir"
}
if [ -z "$profile" ]; then
    case "${SHELL:-}" in
        */bash) if [ "$system" = darwin ]; then profile=$HOME/.bash_profile; else profile=$HOME/.bashrc; fi ;;
        */zsh) profile=$HOME/.zshrc ;;
        */sh) profile=$HOME/.profile ;;
        *) printf '%s\n' 'Unknown shell: add the PATH command manually.'; finish_install; exit 0 ;;
    esac
fi
case ":$PATH:" in *":$install_dir:"*) finish_install; exit 0 ;; esac
if [ -f "$receipt/profile" ]; then finish_install; exit 0; fi
case "$profile" in /*) ;; *) fail 'Profile must be absolute.' ;; esac
case "$profile$install_dir" in *'
'*) fail 'Newlines are not supported in install/profile paths.' ;; esac
[ ! -L "$profile" ] || fail 'Profile cannot be a symlink.'
printf "\n: 'SSH KeyFlow PATH begin'\ncase \":\$PATH:\" in *:'%s':*) ;; *) export PATH='%s':\"\$PATH\" ;; esac\n: 'SSH KeyFlow PATH end'\n" "$quoted_dir" "$quoted_dir" > "$stage/stanza"
printf 'Proposed addition to %s:\n' "$profile"
cat "$stage/stanza"
printf 'Persist this PATH addition? [y/N] '
answer=
if [ "$add_to_path" = true ]; then answer=yes
elif [ -t 1 ] && [ -r /dev/tty ]; then read -r answer < /dev/tty || true
fi
answer=$(printf '%s' "$answer" | tr -d '\r')
case "$answer" in
    y|Y|yes|YES)
        if [ -f "$profile" ] && grep -F 'SSH KeyFlow PATH begin' "$profile" >/dev/null; then fail 'Unowned PATH marker exists; use manual guidance.'; fi
        mkdir -p "$(dirname "$profile")"
        prepare_profile "$profile"
        cat "$stage/stanza" >> "$profile_stage/replacement"
        profile_replaced=true
        mv -f "$profile_stage/replacement" "$profile"
        cp "$stage/stanza" "$receipt/stanza"
        printf '%s\n' "$profile" > "$receipt/profile"
        printf '%s\n' 'PATH saved. Open a new shell or source your profile.' ;;
    *) printf '%s\n' 'Profile unchanged. Use the manual PATH command when needed.' ;;
esac
finish_install
