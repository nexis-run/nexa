#!/usr/bin/env bash

set -euo pipefail

GITHUB_REPO='nexis-run/nexa'
BINARY_NAME='nexa'
INSTALL_TEMP_DIR=''
INSTALL_STAGED_DIR=''
INSTALL_TARGET=''
INSTALL_VERSION=''
INSTALL_FORCE=0

print_info() {
    printf '[INFO] %s\n' "$1"
}

print_warn() {
    printf '[WARN] %s\n' "$1" >&2
}

print_error() {
    printf '[ERROR] %s\n' "$1" >&2
}

cleanup() {
    if [[ -n "$INSTALL_TEMP_DIR" && -d "$INSTALL_TEMP_DIR" ]]; then
        rm -rf -- "$INSTALL_TEMP_DIR"
    fi
    if [[ -n "$INSTALL_STAGED_DIR" && -d "$INSTALL_STAGED_DIR" ]]; then
        rm -rf -- "$INSTALL_STAGED_DIR"
    fi
}

usage() {
    printf '%s\n' \
        '用法：bash install.sh [--version vX.Y.Z] [--force]' \
        '默认安装最新稳定版本，使用 NEXA_INSTALL_DIR 指定安装目录。' \
        '--version 指定稳定发布版本。' \
        '--force 允许重新安装、降级或覆盖无法识别版本的现有程序。'
}

parse_args() {
    while (($# > 0)); do
        case "$1" in
            --version)
                if (($# < 2)); then
                    print_error '--version 缺少版本号'
                    return 1
                fi
                INSTALL_VERSION=$2
                shift 2
                ;;
            --force)
                INSTALL_FORCE=1
                shift
                ;;
            --help | -h)
                usage
                exit 0
                ;;
            *)
                print_error "不支持的参数：$1"
                return 1
                ;;
        esac
    done
}

download() {
    curl -fsSL --proto '=https' --tlsv1.2 --retry 3 --connect-timeout 15 --max-time 300 "$@"
}

read_version() {
    local executable=$1
    local version_output

    version_output=$("$executable" --version 2>/dev/null) || version_output=''
    if [[ -z "$version_output" ]]; then
        version_output=$("$executable" version 2>/dev/null) || version_output=''
    fi

    printf '%s\n' "$version_output" \
        | sed -nE 's/^(nexa version )?v?([^[:space:]]+)([[:space:]].*)?$/\2/p' \
        | head -n 1
}

is_stable_version() {
    [[ "${1#v}" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]
}

version_base() {
    local version=${1#v}

    if [[ "$version" =~ ^((0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*))([-+][0-9A-Za-z.+-]+|\.[0-9a-fA-F]+)?$ ]]; then
        printf '%s\n' "${BASH_REMATCH[1]}"
        return
    fi

    return 1
}

version_lt() {
    local left=$1
    local right=$2
    local left_part right_part

    while [[ -n "$left" ]]; do
        left_part=${left%%.*}
        right_part=${right%%.*}
        if ((${#left_part} != ${#right_part})); then
            ((${#left_part} < ${#right_part}))
            return
        fi
        if [[ "$left_part" != "$right_part" ]]; then
            [[ "$left_part" < "$right_part" ]]
            return
        fi
        if [[ "$left" != *.* ]]; then
            return 1
        fi
        left=${left#*.}
        right=${right#*.}
    done

    return 1
}

get_latest_tag() {
    local latest_tag
    latest_tag=$(download \
        -H 'Accept: application/vnd.github+json' \
        -H 'X-GitHub-Api-Version: 2022-11-28' \
        "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" \
        | sed -nE 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/p' \
        | head -n 1)

    if [[ -z "$latest_tag" ]]; then
        print_error '无法获取最新发布版本，请检查 GitHub 连接，或使用 --version 指定版本'
        return 1
    fi

    printf '%s\n' "$latest_tag"
}

detect_platform() {
    local operating_system
    local architecture

    operating_system=$(uname -s | tr '[:upper:]' '[:lower:]')
    architecture=$(uname -m)

    case "$operating_system" in
        linux*) operating_system='linux' ;;
        darwin*) operating_system='darwin' ;;
        mingw* | msys* | cygwin*) operating_system='windows' ;;
        *)
            print_error "不支持的操作系统：${operating_system}"
            return 1
            ;;
    esac

    case "$architecture" in
        x86_64 | amd64) architecture='amd64' ;;
        arm64 | aarch64) architecture='arm64' ;;
        *)
            print_error "不支持的处理器架构：${architecture}"
            return 1
            ;;
    esac

    printf '%s-%s\n' "$operating_system" "$architecture"
}

resolve_install_dir() {
    local platform=$1
    local install_dir=${NEXA_INSTALL_DIR:-}

    if [[ -z "$install_dir" ]] && command -v go >/dev/null 2>&1; then
        install_dir=$(go env GOBIN)
        if [[ -z "$install_dir" ]]; then
            install_dir=$(go env GOPATH)
            if [[ "$platform" == windows-* ]]; then
                install_dir=${install_dir%%;*}
            else
                install_dir=${install_dir%%:*}
            fi
            install_dir="${install_dir}/bin"
        fi
    fi

    install_dir=${install_dir:-"${HOME}/.local/bin"}
    if [[ "$platform" == windows-* ]] && command -v cygpath >/dev/null 2>&1; then
        install_dir=$(cygpath -u "$install_dir")
    fi
    if [[ "$install_dir" != /* ]]; then
        install_dir="${PWD}/${install_dir}"
    fi

    printf '%s\n' "${install_dir%/}"
}

checksum_file() {
    local path=$1

    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$path" | awk '{print $1}'
        return
    fi
    if command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$path" | awk '{print $1}'
        return
    fi

    print_error '系统中未找到 SHA-256 校验工具'
    return 1
}

install_binary() {
    local tag=$1
    local platform=$2
    local install_dir=$3
    local binary_file="${BINARY_NAME}-${platform}"
    local release_url="https://github.com/${GITHUB_REPO}/releases/download/${tag}"

    if [[ "$platform" == windows-* ]]; then
        binary_file="${binary_file}.exe"
    fi

    INSTALL_TEMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/nexa-install.XXXXXX")
    local downloaded_binary="${INSTALL_TEMP_DIR}/${binary_file}"
    local checksums="${INSTALL_TEMP_DIR}/checksums.txt"

    print_info "下载 ${BINARY_NAME} ${tag}（${platform}）"
    if ! download -o "$checksums" "${release_url}/checksums.txt"; then
        print_error "无法下载 ${tag} 的 checksums.txt，请选择包含完整校验文件的稳定发布版本"
        return 1
    fi
    download -o "$downloaded_binary" "${release_url}/${binary_file}"

    local expected_checksum
    local actual_checksum
    expected_checksum=$(awk -v filename="$binary_file" '$2 == filename || $2 == "*" filename {print $1}' "$checksums")
    if [[ ! "$expected_checksum" =~ ^[0-9a-f]{64}$ ]]; then
        print_error "校验文件中 ${binary_file} 的记录缺失、重复或无效"
        return 1
    fi

    actual_checksum=$(checksum_file "$downloaded_binary")
    if [[ "$actual_checksum" != "$expected_checksum" ]]; then
        print_error "${binary_file} 的 SHA-256 校验失败"
        return 1
    fi

    chmod 0755 "$downloaded_binary"
    local downloaded_version
    downloaded_version=$(read_version "$downloaded_binary")
    if [[ "$downloaded_version" != "${tag#v}" ]]; then
        print_error "下载的程序无法运行或版本不匹配，期望 ${tag#v}，实际 ${downloaded_version:-未知}，现有程序未被替换"
        return 1
    fi

    mkdir -p "$install_dir"
    INSTALL_STAGED_DIR=$(mktemp -d "${install_dir}/.nexa-install.XXXXXX")
    local staged_target="${INSTALL_STAGED_DIR}/${INSTALL_TARGET##*/}"
    install -m 0755 "$downloaded_binary" "$staged_target"
    mv -f -- "$staged_target" "$INSTALL_TARGET"

    print_info "已安装 ${downloaded_version} 到 ${INSTALL_TARGET}"
}

report_path() {
    local install_dir=$1
    local active_binary

    case ":$PATH:" in
        *":${install_dir}:"*) ;;
        *) print_warn "请将 ${install_dir} 加入 PATH" ;;
    esac
    active_binary=$(command -v "$BINARY_NAME" || true)
    if [[ -n "$active_binary" && "$active_binary" != "$INSTALL_TARGET" ]]; then
        print_warn "PATH 当前优先使用 ${active_binary}，本次安装目标为 ${INSTALL_TARGET}"
    fi
}

main() {
    parse_args "$@"
    trap cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM

    local platform
    local install_dir
    local latest_tag
    local latest_version
    platform=$(detect_platform)
    install_dir=$(resolve_install_dir "$platform")
    INSTALL_TARGET="${install_dir}/${BINARY_NAME}"
    if [[ "$platform" == windows-* ]]; then
        INSTALL_TARGET="${INSTALL_TARGET}.exe"
    fi
    if [[ -d "$INSTALL_TARGET" ]]; then
        print_error "安装目标是目录：${INSTALL_TARGET}"
        return 1
    fi

    if [[ -n "$INSTALL_VERSION" ]]; then
        latest_tag="v${INSTALL_VERSION#v}"
    else
        latest_tag=$(get_latest_tag)
    fi
    if ! is_stable_version "$latest_tag"; then
        print_error "发布版本 ${latest_tag} 不支持安全自动安装，请在 https://github.com/${GITHUB_REPO}/releases 选择包含 checksums.txt 的 vX.Y.Z 版本，再用 --version 指定"
        return 1
    fi
    latest_version=${latest_tag#v}

    if [[ -e "$INSTALL_TARGET" || -L "$INSTALL_TARGET" ]] && ((INSTALL_FORCE == 0)); then
        local installed_version
        local installed_base
        installed_version=$(read_version "$INSTALL_TARGET")
        if ! installed_base=$(version_base "$installed_version"); then
            print_error "无法识别 ${INSTALL_TARGET} 的版本，程序保持不变；确认覆盖时请使用 --force"
            return 1
        fi
        if version_lt "$latest_version" "$installed_base"; then
            print_error "目标版本 ${latest_version} 低于已安装版本 ${installed_version}，确认降级时请使用 --force"
            return 1
        fi
        if [[ "${installed_version%%+*}" == "$latest_version" ]]; then
            print_info "${INSTALL_TARGET} 已安装版本 ${installed_version}"
            report_path "$install_dir"
            return
        fi
    fi

    install_binary "$latest_tag" "$platform" "$install_dir"
    report_path "$install_dir"
}

if [[ "${BASH_SOURCE[0]:-$0}" == "$0" ]]; then
    main "$@"
fi
