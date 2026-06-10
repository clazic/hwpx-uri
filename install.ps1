<#
.SYNOPSIS
  hwpx-uri 설치 스크립트 (Windows / PowerShell)

.DESCRIPTION
  빠른 설치 (Claude Code):
    irm https://raw.githubusercontent.com/clazic/hwpx-uri/main/install.ps1 | iex

  옵션 지정 시 (다운로드 후 실행):
    .\install.ps1 -Target all
    .\install.ps1 -DesktopZip "$HOME\Desktop\hwpx-uri.zip"
    .\install.ps1 -Dir "D:\skills"
    .\install.ps1 -Uninstall -Target all

.PARAMETER Target
  claude-code (기본) | codex | all

.PARAMETER Dir
  임의 폴더에 설치 (<Dir>\hwpx-uri)

.PARAMETER DesktopZip
  Claude Desktop/claude.ai 업로드용 .zip 생성 (리눅스 바이너리 포함)

.PARAMETER Ref
  설치할 git 참조 (기본 main)

.PARAMETER Uninstall
  설치 제거
#>
[CmdletBinding()]
param(
  [ValidateSet('claude-code','codex','all')]
  [string]$Target = 'claude-code',
  [string]$Dir,
  [string]$DesktopZip,
  [string]$Ref = 'main',
  [switch]$Uninstall
)

$ErrorActionPreference = 'Stop'
$Repo      = 'clazic/hwpx-uri'
$SkillName = 'hwpx-uri'
$RawBase   = if ($env:HWPX_RAW_BASE) { $env:HWPX_RAW_BASE } else { "https://raw.githubusercontent.com/$Repo/$Ref" }

$CommonFiles = @(
  'SKILL.md',
  'references/form-rules.json',
  'references/form-template.hwpx',
  'schema/ir-schema.json',
  'examples/demo.json'
)

function Log($m) { Write-Host "[hwpx-uri] $m" -ForegroundColor Cyan }
function Fail($m) { Write-Host "[오류] $m" -ForegroundColor Red; exit 1 }

function Get-TargetDir($t) {
  $base = if ($env:CLAUDE_CONFIG_DIR) { $env:CLAUDE_CONFIG_DIR } else { Join-Path $HOME '.claude' }
  switch ($t) {
    'claude-code' { Join-Path $base "skills\$SkillName" }
    'codex'       { Join-Path $HOME ".codex\skills\$SkillName" }
  }
}

function Fetch($url, $out) {
  try { Invoke-WebRequest -Uri $url -OutFile $out -UseBasicParsing }
  catch { Fail "다운로드 실패: $url" }
}

function Stage-Skill($stage, $binary) {
  foreach ($d in @('bin','references','schema','examples')) {
    New-Item -ItemType Directory -Force -Path (Join-Path $stage $d) | Out-Null
  }
  foreach ($f in $CommonFiles) {
    Log "다운로드: $f"
    $out = Join-Path $stage ($f -replace '/','\')
    Fetch "$RawBase/$f" $out
  }
  Log "다운로드: bin/$binary"
  # 로컬(Claude Code/Codex)은 .exe, 클라우드 zip(리눅스)은 확장자 없음
  $binOut = if ($binary -like '*.exe') { Join-Path $stage 'bin\hwpxgen.exe' } else { Join-Path $stage 'bin\hwpxgen' }
  Fetch "$RawBase/bin/$binary" $binOut
}

function Smoke-Test($stage) {
  $exe = Join-Path $stage 'bin\hwpxgen.exe'
  if (-not (Test-Path $exe)) { return }   # 리눅스 바이너리 스테이지는 Windows에서 실행 불가 → 검증 생략
  $tmp = Join-Path ([System.IO.Path]::GetTempPath()) 'hwpx-demo.hwpx'
  & $exe -in (Join-Path $stage 'examples\demo.json') -o $tmp 2>$null | Out-Null
  if ((Test-Path $tmp) -and ((Get-Item $tmp).Length -gt 0)) { Log "✅ 설치 검증 통과 (데모 변환 성공)" }
  else { Write-Host "[경고] 데모 변환 검증 실패 — 아키텍처/실행 권한 확인" -ForegroundColor Yellow }
}

# ---- 제거 모드 ----
if ($Uninstall) {
  $targets = if ($Target -eq 'all') { @('claude-code','codex') } else { @($Target) }
  foreach ($t in $targets) {
    $d = if ($Dir) { Join-Path $Dir $SkillName } else { Get-TargetDir $t }
    if (Test-Path $d) { Remove-Item -Recurse -Force $d; Log "제거됨: $d" } else { Log "없음(건너뜀): $d" }
  }
  exit 0
}

# ---- 스테이징 (호스트=Windows x64) ----
$tmpRoot = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
$Stage = Join-Path $tmpRoot $SkillName
Stage-Skill $Stage 'hwpxgen-win-x64.exe'
Smoke-Test $Stage

# ---- Claude Desktop 업로드용 zip (리눅스 바이너리) ----
if ($DesktopZip) {
  $dTmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
  $dStage = Join-Path $dTmp $SkillName
  Stage-Skill $dStage 'hwpxgen-linux-x64'   # 클라우드 VM 은 Linux
  if (Test-Path $DesktopZip) { Remove-Item -Force $DesktopZip }
  Compress-Archive -Path $dStage -DestinationPath $DesktopZip
  Log "✅ Claude Desktop 업로드용 zip 생성: $DesktopZip"
  Log "   → Claude Desktop/claude.ai: 설정 > 기능(Features) > 스킬 업로드에서 이 zip 을 추가하세요."
}

# ---- 파일시스템 설치 ----
function Install-To($dest) {
  if (Test-Path $dest) { Remove-Item -Recurse -Force $dest }
  New-Item -ItemType Directory -Force -Path (Split-Path $dest) | Out-Null
  Copy-Item -Recurse -Force $Stage $dest
  Log "✅ 설치: $dest"
}

if ($Dir) {
  Install-To (Join-Path $Dir $SkillName)
} elseif (-not $DesktopZip) {
  $targets = if ($Target -eq 'all') { @('claude-code','codex') } else { @($Target) }
  foreach ($t in $targets) { Install-To (Get-TargetDir $t) }
}

Log "완료. Claude Code/Codex 를 재시작하면 'hwpx-uri' 스킬이 인식됩니다."
