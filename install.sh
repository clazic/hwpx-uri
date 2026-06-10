#!/usr/bin/env bash
# hwpx-uri 설치 스크립트 (macOS / Linux)
#
# 빠른 설치 (Claude Code):
#   curl -fsSL https://raw.githubusercontent.com/clazic/hwpx-uri/main/install.sh | bash
#
# 옵션:
#   --target claude-code   ~/.claude/skills/hwpx-uri (기본, CLAUDE_CONFIG_DIR 존중)
#   --target codex         ~/.codex/skills/hwpx-uri
#   --target all           위 둘 다
#   --dir <PATH>           임의 폴더에 설치 (<PATH>/hwpx-uri)
#   --desktop-zip [PATH]   Claude Desktop/claude.ai 업로드용 .zip 생성(리눅스 바이너리 포함)
#   --ref <BRANCH/TAG>     설치할 git 참조 (기본 main)
#   --uninstall            설치 제거
#   -h, --help             도움말
#
# 예:
#   curl -fsSL .../install.sh | bash -s -- --target all
#   curl -fsSL .../install.sh | bash -s -- --desktop-zip ~/Desktop/hwpx-uri.zip
set -euo pipefail

REPO="clazic/hwpx-uri"
REF="main"
SKILL_NAME="hwpx-uri"
TARGETS=()
EXPLICIT_DIR=""
DESKTOP_ZIP=""
DO_UNINSTALL=0

# 기본 소스는 raw.githubusercontent. HWPX_RAW_BASE 로 덮어쓰면 포크·로컬 테스트 가능.
RAW() { echo "${HWPX_RAW_BASE:-https://raw.githubusercontent.com/${REPO}/${REF}}/$1"; }

# 런타임에 필요한 파일(바이너리 제외). 모두 ASCII 경로.
COMMON_FILES=(
  "SKILL.md"
  "references/form-rules.json"
  "references/form-template.hwpx"
  "schema/ir-schema.json"
  "examples/demo.json"
)

log()  { printf '\033[36m[hwpx-uri]\033[0m %s\n' "$*"; }
err()  { printf '\033[31m[오류]\033[0m %s\n' "$*" >&2; }
die()  { err "$*"; exit 1; }

usage() { sed -n '2,28p' "$0" 2>/dev/null || echo "install.sh — hwpx-uri 설치"; exit 0; }

# ---- 인자 파싱 ----
while [ $# -gt 0 ]; do
  case "$1" in
    --target) shift; case "${1:-}" in
                claude-code) TARGETS+=("claude-code");;
                codex)       TARGETS+=("codex");;
                all)         TARGETS+=("claude-code" "codex");;
                *) die "알 수 없는 target: ${1:-}";;
              esac;;
    --dir)         shift; EXPLICIT_DIR="${1:-}";;
    --desktop-zip) # 다음 인자가 옵션이 아니면 경로로 사용, 아니면 기본 경로
                   if [ $# -ge 2 ] && [ "${2#-}" = "$2" ]; then DESKTOP_ZIP="$2"; shift; else DESKTOP_ZIP="__default__"; fi;;
    --ref)         shift; REF="${1:-main}";;
    --uninstall)   DO_UNINSTALL=1;;
    -h|--help)     usage;;
    *) die "알 수 없는 옵션: $1 (--help 참고)";;
  esac
  shift
done

# ---- 플랫폼 → 바이너리 매핑 ----
detect_binary() {
  local os arch
  os=$(uname -s); arch=$(uname -m)
  case "$os" in
    Darwin) case "$arch" in
              arm64) echo "hwpxgen-mac-arm64";;
              x86_64) echo "hwpxgen-mac-intel";;
              *) die "지원하지 않는 macOS 아키텍처: $arch";;
            esac;;
    Linux)  case "$arch" in
              x86_64|amd64) echo "hwpxgen-linux-x64";;
              aarch64|arm64) echo "hwpxgen-linux-arm64";;
              *) die "지원하지 않는 Linux 아키텍처: $arch";;
            esac;;
    *) die "지원하지 않는 OS: $os (Windows 는 install.ps1 사용)";;
  esac
}

# ---- 다운로드 헬퍼 ----
fetch() {
  local url="$1" out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out" || die "다운로드 실패: $url"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url" || die "다운로드 실패: $url"
  else
    die "curl 또는 wget 이 필요합니다"
  fi
}

# ---- 스테이징 폴더에 스킬 구성 ----
stage_skill() {
  local stage="$1" binary="$2"
  mkdir -p "$stage"/{bin,references,schema,examples}
  local f
  for f in "${COMMON_FILES[@]}"; do
    log "다운로드: $f"
    fetch "$(RAW "$f")" "$stage/$f"
  done
  log "다운로드: bin/$binary → bin/hwpxgen"
  fetch "$(RAW "bin/$binary")" "$stage/bin/hwpxgen"
  chmod +x "$stage/bin/hwpxgen"
  # macOS Gatekeeper 격리 속성 제거(있을 때만)
  if [ "$(uname -s)" = "Darwin" ]; then
    xattr -dr com.apple.quarantine "$stage/bin/hwpxgen" 2>/dev/null || true
  fi
}

# ---- 설치 검증(데모 변환) ----
smoke_test() {
  local dir="$1" out
  out="$(mktemp -d)/demo.hwpx"
  if "$dir/bin/hwpxgen" -in "$dir/examples/demo.json" -o "$out" >/dev/null 2>&1 && [ -s "$out" ]; then
    log "✅ 설치 검증 통과 (데모 변환 성공)"
  else
    err "⚠ 데모 변환 검증 실패 — 바이너리 실행 권한/아키텍처를 확인하세요"
  fi
}

resolve_target_dir() {
  case "$1" in
    claude-code) echo "${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/$SKILL_NAME";;
    codex)       echo "$HOME/.codex/skills/$SKILL_NAME";;
  esac
}

# ---- 제거 모드 ----
if [ "$DO_UNINSTALL" = "1" ]; then
  [ ${#TARGETS[@]} -eq 0 ] && TARGETS=("claude-code" "codex")
  for t in "${TARGETS[@]}"; do
    d="$(resolve_target_dir "$t")"
    [ -n "$EXPLICIT_DIR" ] && d="$EXPLICIT_DIR/$SKILL_NAME"
    if [ -d "$d" ]; then rm -rf "$d"; log "제거됨: $d"; else log "없음(건너뜀): $d"; fi
  done
  exit 0
fi

BINARY="$(detect_binary)"
log "플랫폼 바이너리: $BINARY (ref=$REF)"
STAGE="$(mktemp -d)/$SKILL_NAME"
stage_skill "$STAGE" "$BINARY"
smoke_test "$STAGE"

# ---- Claude Desktop 업로드용 zip (리눅스 바이너리) ----
if [ -n "$DESKTOP_ZIP" ]; then
  command -v zip >/dev/null 2>&1 || die "zip 명령이 필요합니다"
  ZOUT="$DESKTOP_ZIP"; [ "$ZOUT" = "__default__" ] && ZOUT="$PWD/hwpx-uri-claude-desktop.zip"
  DSTAGE="$(mktemp -d)/$SKILL_NAME"
  stage_skill "$DSTAGE" "hwpxgen-linux-x64"   # 클라우드 VM 은 Linux
  ( cd "$(dirname "$DSTAGE")" && zip -qr "$ZOUT" "$SKILL_NAME" )
  log "✅ Claude Desktop 업로드용 zip 생성: $ZOUT"
  log "   → Claude Desktop/claude.ai: 설정 > 기능(Features) > 스킬 업로드에서 이 zip 을 추가하세요."
fi

# ---- 파일시스템 설치(Claude Code / Codex) ----
if [ -n "$EXPLICIT_DIR" ]; then
  dest="$EXPLICIT_DIR/$SKILL_NAME"
  rm -rf "$dest"; mkdir -p "$(dirname "$dest")"; cp -R "$STAGE" "$dest"
  log "✅ 설치: $dest"
elif [ ${#TARGETS[@]} -gt 0 ]; then
  for t in "${TARGETS[@]}"; do
    dest="$(resolve_target_dir "$t")"
    rm -rf "$dest"; mkdir -p "$(dirname "$dest")"; cp -R "$STAGE" "$dest"
    log "✅ 설치($t): $dest"
  done
elif [ -z "$DESKTOP_ZIP" ]; then
  # 기본: Claude Code
  dest="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/$SKILL_NAME"
  rm -rf "$dest"; mkdir -p "$(dirname "$dest")"; cp -R "$STAGE" "$dest"
  log "✅ 설치(claude-code): $dest"
fi

log "완료. Claude Code/Codex 를 재시작하면 'hwpx-uri' 스킬이 인식됩니다."
