---
name: hwpx-uri
description: 울산연구원 보고서 전용 HWPX 생성 스킬(Go 단일 바이너리). 작성가이드 규칙으로 쓴 마크다운(.md) 또는 IR(JSON)을 「울산연구원 기본정책과제 보고서」 양식 HWPX로 변환한다. 흐름도·차트 이미지를 직접 렌더링하고 각주·표·그림·미니목차를 자동 생성한다. 양식 적응형·자기완결. 울산연구원/기본정책과제/정책보고서/한글 보고서 키워드에서 활성화.
---

# 울산연구원 보고서 HWPX 생성 스킬 (Go)

## 0. 정체성

「울산연구원 기본정책과제 보고서」를 HWPX(.hwpx)로 생성하는 **전용 스킬**이다.
입력 해석(보고서→계층 판단)은 사람/LLM이 IR로, **조립은 Go 바이너리가 결정론적으로** 수행한다.
정규식으로 기호를 추측하지 않고 마크다운 헤딩·IR kind로 계층을 **명시**하므로 계층 오인식이 없다.

## 1. 빌드 (최초 1회)

### 이 스킬이 실행하는 바이너리: `bin/hwpxgen`
스킬은 스킬 폴더의 `bin/hwpxgen` 을 실행한다. Go 소스는 `src/` 에 있고, 현재 환경용으로 한 번만 빌드한다:
```bash
cd ~/.claude/skills/hwpx-uri/src
go build -o ../bin/hwpxgen .
```
빌드엔 **Go 1.21+** 만 필요하며, 한 번 만들면 이후 변환에는 Go 가 필요 없다.
(Windows 는 `go build -o ..\bin\hwpxgen.exe .`)

### 크로스 컴파일 — macOS + Windows 동시 (★ 권장)
한 대의 머신(맥이든 윈도우든)에서 **모든 플랫폼 바이너리를 한 번에** 만든다. 순수 Go(cgo 없음)라 가능하다.

```bash
src/build.sh      # macOS / Linux 에서 실행
src\build.bat     # Windows 에서 실행 (더블클릭 또는 cmd)
```

산출물 `bin/`:
| 파일 | 대상 |
|---|---|
| `hwpxgen-mac-arm64` | macOS Apple Silicon (M1~) |
| `hwpxgen-mac-intel` | macOS Intel |
| `hwpxgen-win-x64.exe` | Windows 64bit |

- 필요한 것: 빌드 시 **Go 1.21+** 하나뿐. Python·venv 불필요.
- 의존성: `beevik/etree`(XML round-trip 보존), `golang.org/x/image`(이미지 렌더). `go build`가 자동 설치.
- 빌드는 **개발자 한 명만 1회** 하면 되고, 산출된 실행파일은 Go 설치 없이 어디서나 돈다.

### 배포 (실행 PC — Go 설치 불필요)
실행 PC에는 **실행파일 1개 + `references/` 폴더**만 같이 두면 된다.

```
배포폴더/
├── hwpxgen(-mac-arm64 | -mac-intel | -win-x64.exe)   ← 자기 OS용 1개
└── references/
    ├── form-template.hwpx  ← 기본 템플릿 (구 양식_통합)
    └── form-rules.json
```

- macOS: `hwpxgen-mac-arm64` (또는 intel) 사용. 최초 실행 시 Gatekeeper 차단되면 **우클릭 → 열기** 1회 또는 `xattr -dr com.apple.quarantine hwpxgen-mac-arm64`.
- Windows: `hwpxgen-win-x64.exe` 사용. SmartScreen 경고 시 **추가 정보 → 실행**.
- **한글 폰트**: 흐름도·차트 이미지는 OS 기본 한글 폰트를 자동 탐색한다 — Windows `맑은 고딕(malgun.ttf)`, macOS `AppleGothic`. 둘 다 OS 기본 설치라 별도 작업 불필요.

## 2. 사용

```bash
BIN=~/.claude/skills/hwpx-uri/bin/hwpxgen   # (Windows: ...\bin\hwpxgen.exe)

# 마크다운 보고서 → HWPX (작성가이드 규칙으로 작성한 .md)
$BIN -in 보고서.md -o 보고서.hwpx

# IR(JSON) → HWPX (정밀 제어용)
$BIN -in 보고서.json -o 보고서.hwpx
```

변환 후 자동 검증(`[validate] ✅ 검증 통과`)이 실행된다. 기본 양식·규칙은 `references/`에서 찾는다.

> 위 예시의 `hwpxgen` 은 단일 빌드 이름이다. 크로스 컴파일 배포본은 OS별 이름을 쓴다:
> macOS `./hwpxgen-mac-arm64 -in 보고서.md -o 보고서.hwpx`, Windows `hwpxgen-win-x64.exe -in 보고서.md -o 보고서.hwpx`.

## 3. 입력 ① 마크다운 (작성가이드 규칙)

마크다운 **헤딩 레벨로 계층을 명시**한다. 번호(Ⅰ. 1. 1) (1))는 써도 되지만 **조립기가 자동 재부여**하므로 중복되지 않는다(번호는 신경 쓰지 말 것).

```markdown
---
title: 울산형 청년주거 정책방향 연구
series: 기본정책과제 2026-01
lab: 도시공간연구실
author: 김울산
date: 2026년 12월
director: 표상훈
---

# 요약
청년 주거불안에 대응하는 울산형 정책 방향을 제시함

# Ⅰ. 청년주거 현황과 진단      ← H1 = 장
## 1. 청년 인구·가구 구조 변화    ← H2 = 절
### 1) 1인가구의 급증            ← H3 = 세부
#### (1) 연령대별 분포           ← H4 = 상세
○ 청년 1인가구가 빠르게 증가함    ← 주요 항목
  – 최근 5년간 연평균 4.2% 증가  ← 세부 설명(en dash)
  • 통계청(2026)                ← 참고/출처
※ 1인 청년가구 공급은 제한적      ← 비고

**[표 1-1]  청년 가구 구조 변화**   ← 표 캡션(번호는 자동)
| 구분 | 2020 | 2025 |
|------|------|------|
| 1인가구 | 35% | 48% |
*자료: 통계청(2026)*

![캡션](그림.png)                 ← 외부 이미지

```flow                          ← 흐름도(자동 렌더링)
제도 검토 | 실태 분석 | 사례 비교    ← '|' 있는 줄 = 상단 병렬 박스
쟁점 도출
정책 제언
\```

본문에 각주 마커[^1] 사용              ← 각주
[^1]: 각주 내용

# 참고문헌
• 저자(2025). 『제목』. 발행기관.

# 부록
## 1. 부록 제목
○ 부록 내용
```

| 마크다운 | 보고서 요소 |
|---|---|
| frontmatter `title/series/lab/author/date/director` | 표지 메타 (한글 키 `제목/과제번호/연구실/저자/발간월/원장` 도 지원) |
| `# 발간사` | 발간사 본문 (생략 시 해당 영역 비움) |
| `# 요약` | 요약 4블록 — 양식 고정 라벨 **연구목적 및 필요성 / 연구 주요 내용 / 결론 및 정책제언 / 정책 활용실적 및 계획**에 순서대로 매핑 (생략 블록은 비움) |
| `# Ⅰ.` / `## 1.` / `### 1)` / `#### (1)` | 장/절/세부/상세 (번호 자동) |
| `○` / `– `/`- ` / `• ` / `※ ` | 주요/세부설명/참고/비고 |
| `**[표 N-M] 제목**` + `\| 표 \|` + `*자료:*` | 표 세트 (번호 자동) |
| `![](경로)` | 외부 이미지 그림 |
| ` ```flow ` 블록 | 흐름도 이미지 자동 생성 |
| `[^N]` + `[^N]: 내용` | 각주 |
| `# 참고문헌` / `# 부록` | 참고문헌 / 부록 |
| `# 목차` | 무시(자동 생성 예정) |

## 4. 입력 ② IR(JSON)

마크다운으로 표현하기 어려운 차트나 정밀 제어가 필요하면 IR(JSON)을 직접 작성한다.
스키마는 [schema/ir-schema.json](schema/ir-schema.json), 예시는 [examples/demo.json](examples/demo.json).

차트(막대그래프)는 현재 IR JSON 으로만 지정한다:
```json
{ "kind": "figure", "payload": { "figure": {
    "caption": "청년 주거점유 형태 변화",
    "chart": { "labels": ["자가","전세","월세"],
               "series": { "2020년": [15,40,35], "2025년": [12,28,52] } }
}}}
```

## 5. 파이프라인

```
입력(.md / .json)
  → mdparse / ir.Load   : IR(Report) 로 정규화
  → assemble            : 양식 앵커 복제 + 텍스트 치환 (서식은 양식이 소유)
      · 번호 자동부여(장 Ⅰ·절 1.·표 N-M), 쪽나눔, 빈줄
      · 표 재구성, 그림/흐름도/차트 이미지 동적 삽입(BinData+content.hpf)
      · 각주(footNote), 표지 메타, 요약, 발간사
  → validate            : placeholder 잔여·ns0·ZIP 검증
출력 .hwpx
```

## 6. 폴더 구조 / 모듈 구성

```
hwpx-uri/
├── SKILL.md            ← 스킬 정의(이 문서)
├── bin/                ← 실행 바이너리 (스킬이 실행, build 산출물)
├── references/         ← 양식 hwpx·완성본 샘플·form-rules.json (런타임 필요)
├── schema/             ← IR JSON 스키마
├── examples/           ← IR 예시
└── src/                ← Go 개발 소스 (빌드 시에만 필요)
    ├── main.go, go.mod, go.sum
    ├── build.sh, build.bat
    └── internal/
```

| 경로 | 역할 |
|---|---|
| `src/main.go` | CLI 엔트리(.md/.json 분기, 변환, 검증) |
| `src/internal/ir/` | 중간표현(Report/Node) + JSON 로더 |
| `src/internal/mdparse/` | 마크다운(작성가이드) → IR |
| `src/internal/hwpx/` | HWPX ZIP I/O·단락 편집·이미지 등록(etree) |
| `src/internal/profile/` | 양식 스타일 추출·앵커 탐색 |
| `src/internal/rules/` | form-rules.json 로더·번호 형식 |
| `src/internal/assemble/` | IR → section0/1 조립(본문·표·그림·각주·표지) |
| `src/internal/imagegen/` | 흐름도·막대그래프 PNG 렌더(x/image) |
| `src/internal/validate/` | 완성본 검증 |
| `references/` | 양식 hwpx·완성본 샘플·form-rules.json·분석문서 |

## 7. 양식 교체 / references 구성

| 파일 | 역할 |
|---|---|
| `form-template.hwpx` | **기본 템플릿**(구 양식_통합) — 빈 양식에 완성본의 추가 정의(paraPr 92/93·charPr 94)와 그림·절 앵커를 이식한 통합본. placeholder 기반이라 미지정 메타는 검증이 잡아냄. 파일명은 크로스플랫폼 안정성을 위해 ASCII |
| `보고서-양식.hwpx` | 원본 빈 양식(이식 전) — 양식 개정 대조용 보관 |
| `완성본_샘플.hwpx` | 실제 발간 보고서 — 스타일 실측·검증 참조용 보관 |
| `form-rules.json` | 구조 규칙(스타일 ID·기호·번호·쪽나눔) |
| `양식_심층분석보고서.md` | 양식 XML 실측 분석 문서 |

양식이 개정되면 **코드 수정 없이** 두 파일만 교체한다:
1. `references/form-template.hwpx`를 새 버전으로 (필요시 추가 정의 재이식)
2. `references/form-rules.json`의 스타일 ID·paraPr·기호·번호 규칙을 새 양식 실측값으로 갱신

서식(폰트·색·들여쓰기)은 양식이 소유하므로 조립기가 정의하지 않는다. form-rules.json은 구조 규칙(기호·번호·쪽나눔)만 담는다.

## 8. 검증 체크리스트

- [ ] `[validate] ✅ 검증 통과` (placeholder 잔여 0, ns0 0)
- [ ] 한글에서 빈 페이지 없이 열림
- [ ] 표지·요약·본문 계층·표·그림·흐름도·각주·참고문헌 정상
- [ ] 번호 중복(`1. 1.`) 없음, 2번째 절부터 간격 적용

## 9. 한계 / 미구현

| 항목 | 상태 |
|---|---|
| 본문 쪽번호 | **구현 완료** — masterpage2/3 의 `autoNum(PAGE)` 가 본문 쪽번호를 찍는다. 기본값은 **1장 시작 = 1쪽, 이후 연속 번호**(조립기가 2장 이후 간지의 newNum 을 제거). 앞부속(표지~목차)은 쪽번호 없음(masterpage0/1). 완성본 원본처럼 장별 1부터 리셋하려면 form-rules.json `layout.pageNumRestartPerChapter: true` |
| 목차 페이지 숫자(00→실제) | 미구현 — 자동 필드가 아닌 텍스트 더미. 한글 "차례 새로 고침"으로 보완 |
| 간지 미니목차 페이지 숫자 | 미구현 — "00" 더미 텍스트 |
| 목차/표목차/그림목차 재생성 | 미구현 — 목차 placeholder 는 비워서 출력(검증 통과). 제목 자동 기입은 추후 |
| 차트 md 문법 | 미지원 — IR JSON 으로만 |

> 상세 실측 근거: [references/양식_심층분석보고서.md](references/양식_심층분석보고서.md) §11 (완성본 증보)
