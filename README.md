# hwpx-uri

> 울산연구원 「기본정책과제 보고서」 전용 **HWPX 생성 스킬** — Go 단일 바이너리

마크다운(`.md`) 또는 IR(`.json`)로 쓴 보고서를 울산연구원 공식 보고서 양식의 **HWPX(.hwpx)** 로 변환합니다. 입력의 계층 해석은 사람/LLM이, **조립은 Go 바이너리가 결정론적으로** 수행하므로 계층 오인식이 없습니다. 흐름도·막대그래프를 직접 렌더링하고 각주·표·그림·미니목차를 자동 생성합니다.

- **자기완결**: 실행 PC에 Go·Python 불필요 (바이너리 + 양식 파일만)
- **양식 적응형**: 서식은 양식(HWPX)이 소유, 조립기는 단락을 복제해 텍스트만 교체
- **크로스플랫폼**: macOS(arm64/intel) · Linux(x64/arm64) · Windows(x64)

---

## 설치

### Claude Code (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/clazic/hwpx-uri/main/install.sh | bash
```

→ `~/.claude/skills/hwpx-uri/` 에 설치됩니다. Claude Code 를 재시작하면 인식됩니다.

### Claude Code (Windows / PowerShell)

```powershell
irm https://raw.githubusercontent.com/clazic/hwpx-uri/main/install.ps1 | iex
```

→ `%USERPROFILE%\.claude\skills\hwpx-uri\` 에 설치됩니다.

### Codex (CLI)

```bash
curl -fsSL https://raw.githubusercontent.com/clazic/hwpx-uri/main/install.sh | bash -s -- --target codex
```

→ `~/.codex/skills/hwpx-uri/` 에 설치됩니다.

### Claude Code + Codex 둘 다

```bash
curl -fsSL https://raw.githubusercontent.com/clazic/hwpx-uri/main/install.sh | bash -s -- --target all
```

### Claude Desktop / claude.ai (업로드용 zip 생성)

Claude Desktop·claude.ai 는 폴더를 직접 넣을 수 없고 **설정 > 기능(Features)** 에서 스킬 **.zip 을 업로드**합니다. 스킬은 클라우드 **Linux VM** 에서 실행되므로 리눅스 바이너리가 든 zip 을 만들어 업로드하세요.

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/clazic/hwpx-uri/main/install.sh | bash -s -- --desktop-zip ~/Desktop/hwpx-uri.zip
```
```powershell
# Windows
irm https://raw.githubusercontent.com/clazic/hwpx-uri/main/install.ps1 | iex; .\install.ps1 -DesktopZip "$HOME\Desktop\hwpx-uri.zip"
```
생성된 `hwpx-uri.zip` 을 Claude Desktop/claude.ai 의 스킬 업로드에 추가합니다.
> ⚠️ 클라우드 VM 에 한글 폰트가 없으면 **흐름도·차트 이미지의 한글이 깨질 수 있습니다**. 텍스트·표·그림 변환은 정상입니다. (코드 실행 기능이 켜진 Pro/Max/Team/Enterprise 플랜 필요)

### 설치 옵션

| 옵션 | 설명 |
|---|---|
| `--target claude-code\|codex\|all` | 설치 대상 (기본 `claude-code`) |
| `--dir <PATH>` | 임의 폴더에 설치 (`<PATH>/hwpx-uri`) |
| `--desktop-zip [PATH]` | Claude Desktop 업로드용 zip 생성 |
| `--ref <BRANCH/TAG>` | 설치할 git 참조 (기본 `main`) |
| `--uninstall` | 설치 제거 (`--target` 와 함께) |

(PowerShell 은 `-Target`, `-Dir`, `-DesktopZip`, `-Ref`, `-Uninstall`)

### 제거

```bash
curl -fsSL https://raw.githubusercontent.com/clazic/hwpx-uri/main/install.sh | bash -s -- --uninstall --target all
```

---

## 사용법

설치 후 Claude/Codex 에게 **자연어로 요청**하면 스킬이 자동 활성화됩니다:

> "이 마크다운을 울산연구원 기본과제 보고서 HWPX로 만들어줘"

또는 바이너리를 **직접 실행**할 수도 있습니다:

```bash
BIN=~/.claude/skills/hwpx-uri/bin/hwpxgen      # Codex: ~/.codex/skills/...

# 마크다운 보고서 → HWPX
$BIN -in 보고서.md -o 보고서.hwpx

# IR(JSON) → HWPX (정밀 제어용)
$BIN -in 보고서.json -o 보고서.hwpx
```

변환 후 자동 검증(`[validate] ✅ 검증 통과`)이 실행됩니다.

### 마크다운 작성 규칙

**헤딩 레벨로 계층을 명시**합니다. 번호(Ⅰ. 1. 1))는 써도 조립기가 자동 재부여하므로 신경 쓰지 않아도 됩니다.

```markdown
---
title: 울산형 청년주거 정책방향 연구
series: 기본정책과제 2026-01
lab: 도시공간연구실
author: 김울산
date: 2026년 12월
director: 표상훈
---

# 발간사
울산은 산업도시의 성장 과정을 지나 …          ← 발간사 본문(생략 가능)

# 요약
연구목적 및 필요성에 대응하는 정책 방향을 제시함  ← 요약 4블록(순서대로 매핑)

# Ⅰ. 청년주거 현황과 진단      ← H1 = 장(Ⅰ Ⅱ …)
## 1. 청년 인구·가구 구조 변화    ← H2 = 절(1. 2. …)
### 1) 1인가구의 급증            ← H3 = 세부(1) 2))
#### (1) 연령대별 분포           ← H4 = 상세((1) (2))
○ 청년 1인가구가 빠르게 증가함    ← 주요 항목
  – 최근 5년간 연평균 4.2% 증가  ← 세부 설명(en dash)
※ 1인 청년가구 공급은 제한적      ← 비고

**[표 1-1]  청년 가구 구조 변화**   ← 표 캡션(번호 자동)
| 구분 | 2020 | 2025 |
|------|------|------|
| 1인가구 | 35% | 48% |
*자료: 통계청(2026)*

![캡션](그림.png)                 ← 외부 이미지

```flow                          ← 흐름도(자동 렌더링)
제도 검토 | 실태 분석 | 사례 비교
쟁점 도출
정책 제언
\```

본문에 각주 마커[^1] 사용
[^1]: 각주 내용

# 참고문헌
• 저자(2025). 『제목』. 발행기관.

# 부록
## 1. 부록 제목
○ 부록 내용
```

| 마크다운 | 보고서 요소 |
|---|---|
| frontmatter `title/series/lab/author/date/director` (한글 키 `제목/과제번호/연구실/저자/발간월/원장` 도 가능) | 표지 메타 |
| `# 발간사` | 발간사 본문(생략 시 비움) |
| `# 요약` | 요약 4블록 → 연구목적 및 필요성 / 연구 주요 내용 / 결론 및 정책제언 / 정책 활용실적 및 계획 |
| `# Ⅰ.` / `## 1.` / `### 1)` / `#### (1)` | 장 / 절 / 세부 / 상세 (번호 자동) |
| `○` / `– `,`- ` / `※ ` | 주요 / 세부설명 / 비고 |
| `**[표 N-M] 제목**` + 표 + `*자료:*` | 표 세트 (번호 자동) |
| `![](경로)` | 외부 이미지 그림 |
| ` ```flow ` 블록 | 흐름도 이미지 자동 생성 |
| `[^N]` + `[^N]: 내용` | 각주 |
| `# 참고문헌` / `# 부록` | 참고문헌 / 부록 |

차트(막대그래프)는 IR(JSON)로만 지정합니다. 스키마는 [`schema/ir-schema.json`](schema/ir-schema.json), 예시는 [`examples/demo.json`](examples/demo.json) 참고.

---

## 빌드 (개발자용)

실행 바이너리는 `bin/` 에 포함돼 있어 일반 사용자는 빌드가 필요 없습니다. 직접 빌드하려면 **Go 1.21+** 만 있으면 됩니다:

```bash
cd src
go build -o ../bin/hwpxgen .       # 현재 OS용
./build.sh                          # 전 플랫폼 크로스 컴파일 (Windows: build.bat)
```

## 폴더 구조

```
hwpx-uri/
├── SKILL.md            # 스킬 정의
├── install.sh / .ps1   # 설치 스크립트
├── bin/                # 실행 바이너리 (mac/linux/win)
├── references/         # form-template.hwpx · form-rules.json · 분석 문서
├── schema/             # IR JSON 스키마
├── examples/           # IR 예시
└── src/                # Go 소스 (빌드 시에만 필요)
```

## 한계 / 미구현

| 항목 | 상태 |
|---|---|
| 본문 쪽번호 | 양식이 자체 표시. 기본 1장부터 연속 번호 (`form-rules.json` `layout.pageNumRestartPerChapter` 로 장별 리셋 전환 가능) |
| 목차 / 표목차 / 그림목차 페이지 숫자 | 미구현 — 한글 "차례 새로 고침"으로 보완 |
| 간지 미니목차 페이지 숫자 | 미구현 — "00" 더미 |
| 차트 마크다운 문법 | 미지원 — IR JSON 으로만 |
| 클라우드 VM 한글 폰트 | 흐름도·차트 한글 깨질 수 있음 (로컬 실행은 OS 폰트 자동 탐색) |

## 라이선스 / 출처

울산연구원 보고서 양식 기반 내부 도구. 양식 파일(`references/`)의 저작권은 울산연구원에 있습니다.
