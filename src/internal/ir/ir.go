// Package ir 은 변환 파이프라인의 단일 계약(중간표현)을 정의한다.
//
// 설계 철학: 입력 해석(실제 보고서 → 계층 판단)은 LLM 이 담당하고,
// 그 결과를 이 IR(JSON) 로 표현한다. 조립기(assemble)는 오직 이 IR 만 보고
// HWPX 를 만든다. 정규식으로 기호를 "추측"하지 않으므로 계층 오인식이 없다.
//
// JSON 스키마는 schema/ir-schema.json 과 1:1 로 대응한다.
package ir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// NodeKind 는 본문 트리 노드의 종류다. 양식의 계층 스타일과 직접 대응한다.
const (
	KindChapter = "chapter" // 장 (Ⅰ, Ⅱ …) — 간지 + 미니목차 + 장제목으로 전개
	KindSection = "section" // 소절 (1.)
	KindSubsec  = "subsec"  // 세부 (1))
	KindDetail  = "detail"  // 상세 ((1))
	KindPoint   = "point"   // 주요 항목 (○)
	KindSub     = "sub"     // 세부 설명 (-)
	KindNote    = "note"    // 비고 (※)
	KindTable   = "table"   // 표 (Payload.Table)
	KindFigure  = "figure"  // 그림 (Payload.Figure)
	KindPara    = "para"    // 일반 문단 (바탕글)
)

// Meta 는 표지·발간 메타데이터다.
type Meta struct {
	Title    string `json:"title"`            // 보고서 제목
	Series   string `json:"series,omitempty"` // 시리즈 (예: "기본정책과제 2024-01")
	Lab      string `json:"lab"`              // 연구실명
	Author   string `json:"author"`           // 연구자명
	PubDate  string `json:"pub_date"`         // 발간년월 (예: "2026년 12월")
	Director string `json:"director"`         // 울산연구원장명
}

// Summary 는 요약 4블록이다.
type Summary struct {
	Purpose     string `json:"purpose"`     // 연구목적 및 필요성
	Contents    string `json:"contents"`    // 연구 주요 내용
	Conclusion  string `json:"conclusion"`  // 결론 및 정책제언
	Utilization string `json:"utilization"` // 정책 활용실적 및 계획
}

// Table 은 데이터표 payload 다.
type Table struct {
	Title   string     `json:"title,omitempty"`   // 표 제목(캡션). [표 N-M] 번호는 조립기가 자동 부여
	Headers []string   `json:"headers,omitempty"` // 헤더행
	Rows    [][]string `json:"rows,omitempty"`    // 데이터행들
	Source  string     `json:"source,omitempty"`  // 자료 출처
}

// Chart 는 막대그래프 스펙이다(figure payload).
type Chart struct {
	Title  string             `json:"title,omitempty"`
	Labels []string           `json:"labels,omitempty"` // 그룹 라벨(x축)
	Series map[string][]int   `json:"series,omitempty"` // 계열명 → 값들
}

// Flow 는 세로 흐름 순서도 스펙이다(figure payload).
type Flow struct {
	Title       string   `json:"title,omitempty"`
	ParallelTop []string `json:"parallel_top,omitempty"` // 상단 병렬 박스(합류)
	Steps       []string `json:"steps,omitempty"`        // 세로 단계 박스
}

// Figure 는 그림 payload 다. ImagePath / Chart / Flow 중 하나가 채워진다.
type Figure struct {
	Caption   string `json:"caption,omitempty"`    // 그림 제목. [그림 N-M] 번호는 자동 부여
	ImagePath string `json:"image_path,omitempty"` // 실제 이미지 파일 경로
	Chart     *Chart `json:"chart,omitempty"`      // 막대그래프 자동 생성
	Flow      *Flow  `json:"flow,omitempty"`       // 흐름도 자동 생성
}

// Payload 는 table/figure 노드의 부가 데이터다.
type Payload struct {
	Table  *Table  `json:"table,omitempty"`
	Figure *Figure `json:"figure,omitempty"`
}

// Node 는 본문 트리 노드(재귀)다.
//
// Text 에는 양식 기호를 포함하지 않는다(예: "○ 내용" 이 아니라 "내용").
// 기호는 양식 단락이 이미 가지고 있으며, 계층은 Kind 로 명시되기 때문이다.
// 단, 장·절 등의 번호(Ⅰ., 1.)는 조립기가 카운터로 자동 부여한다.
type Node struct {
	Kind     string   `json:"kind"`
	Text     string   `json:"text,omitempty"`
	Children []*Node  `json:"children,omitempty"`
	Payload  *Payload `json:"payload,omitempty"`
}

// Report 는 조립기에 넘기는 최종 정규화 결과다.
type Report struct {
	Meta       Meta              `json:"meta"`
	Summary    Summary           `json:"summary"`
	Foreword   string            `json:"foreword,omitempty"`   // 발간사 본문(선택)
	Body       []*Node           `json:"body"`                 // 장(chapter) 리스트
	References []string          `json:"references,omitempty"` // 참고문헌 항목들
	Appendix   []*Node           `json:"appendix,omitempty"`   // 부록 (절/항목)
	Footnotes  map[string]string `json:"footnotes,omitempty"`  // 각주 {"1": "내용"} — 본문 [^1] 참조
}

// Load 는 JSON 파일에서 Report 를 읽는다.
func Load(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("IR JSON 읽기 실패: %w", err)
	}
	return Parse(data)
}

// Parse 는 JSON 바이트에서 Report 를 디코드하고 기본 검증을 한다.
func Parse(data []byte) (*Report, error) {
	var r Report
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields() // 오타·잘못된 키를 조용히 넘기지 않고 즉시 알린다
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("IR JSON 파싱 실패: %w", err)
	}
	if err := r.validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

// validate 는 IR 의 구조적 정합성을 검사한다(조립 전 조기 실패).
func (r *Report) validate() error {
	var walk func(n *Node, path string) error
	walk = func(n *Node, path string) error {
		if !validKind[n.Kind] {
			return fmt.Errorf("%s: 알 수 없는 kind %q", path, n.Kind)
		}
		if n.Kind == KindTable && (n.Payload == nil || n.Payload.Table == nil) {
			return fmt.Errorf("%s: table 노드에 payload.table 이 없음", path)
		}
		if n.Kind == KindFigure && (n.Payload == nil || n.Payload.Figure == nil) {
			return fmt.Errorf("%s: figure 노드에 payload.figure 가 없음", path)
		}
		for i, c := range n.Children {
			if err := walk(c, fmt.Sprintf("%s>%s[%d]", path, n.Kind, i)); err != nil {
				return err
			}
		}
		return nil
	}
	for i, ch := range r.Body {
		if ch.Kind != KindChapter {
			return fmt.Errorf("body[%d]: 최상위는 chapter 여야 함 (got %q)", i, ch.Kind)
		}
		if err := walk(ch, fmt.Sprintf("body[%d]", i)); err != nil {
			return err
		}
	}
	for i, n := range r.Appendix {
		if err := walk(n, fmt.Sprintf("appendix[%d]", i)); err != nil {
			return err
		}
	}
	return nil
}

var validKind = map[string]bool{
	KindChapter: true, KindSection: true, KindSubsec: true, KindDetail: true,
	KindPoint: true, KindSub: true, KindNote: true,
	KindTable: true, KindFigure: true, KindPara: true,
}
