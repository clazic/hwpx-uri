// Package mdparse 는 작성가이드 규칙으로 쓴 마크다운 보고서를 IR(Report)로 변환한다.
//
// 계층은 마크다운 헤딩 레벨로 "명시"되므로 기호 추측이 없다(오인식 0):
//   # Ⅰ. 챕터  → chapter   (단 # 요약/목차/참고문헌/부록/발간사 는 특수 섹션)
//   ## 1. 절    → section
//   ### 1) 세부 → subsec
//   #### (1) 상세 → detail
//   ○ 항목      → point
//   – / - 항목   → sub
//   • 항목       → sub (참고/출처)
//   ※ 비고       → note
//
// 번호(Ⅰ. 1. 1) (1))는 입력에서 제거하고 조립기가 카운터로 재부여한다(중복 차단).
package mdparse

import (
	"regexp"
	"strings"

	"hwpx-uri-gen/internal/ir"
)

var (
	reHeading   = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	reNumPrefix = regexp.MustCompile(`^(?:[ⅠⅡⅢⅣⅤⅥⅦⅧⅨⅩⅪⅫ]+\.|[IVXLCDM]+\.|\(\d+\)|\d+\)|\d+\.)\s*`)
	reSymPrefix = regexp.MustCompile(`^[○●◦–—\-•※]\s*`)
	reImage     = regexp.MustCompile(`^\s*!\[([^\]]*)\]\(([^)]+)\)\s*$`)
	reCaption   = regexp.MustCompile(`^\s*\*{0,2}\[(?:표|그림)\s*[\d.\-]+\]\s*(.+?)\*{0,2}\s*$`)
	reCaption2  = regexp.MustCompile(`^\s*\*{0,2}(?:표|그림)\s*[:：]\s*(.+?)\*{0,2}\s*$`)
	reSource    = regexp.MustCompile(`^\s*\*?자료\s*[:：]\s*(.+?)\*?\s*$`)
	reFnDef     = regexp.MustCompile(`^\[\^(\w+)\]:\s*(.+)$`)
	reHRule     = regexp.MustCompile(`^\s*(?:-{3,}|\*{3,}|_{3,})\s*$`)
	reFence     = regexp.MustCompile("^```\\s*([A-Za-z]*)\\s*$")
)

// 특수 섹션 라벨(번호·기호 제거 후 비교)
var specialLabels = map[string]string{
	"요약": "summary", "목차": "toc", "표목차": "toc", "그림목차": "toc",
	"참고문헌": "refs", "참고자료": "refs", "부록": "appendix", "발간사": "foreword",
}

// 요약 4블록 라벨(있으면 블록 구분에 사용)
var summaryLabels = []string{"연구목적", "연구 주요", "연구방법", "주요 연구", "결론", "정책 제언", "정책제언", "정책 활용", "연구 배경"}

type parser struct {
	rep            *ir.Report
	stack          []*stackItem
	mode           string // "" | summary | refs | appendix | foreword
	target         *[]*ir.Node
	summaryBlocks  [][]string
	forewordBuf    []string
	pendingCaption string
	pendingSource  string
}

type stackItem struct {
	depth int
	node  *ir.Node
}

var depthOf = map[string]int{
	ir.KindChapter: 1, ir.KindSection: 2, ir.KindSubsec: 3,
	ir.KindDetail: 4, ir.KindPoint: 5, ir.KindSub: 6, ir.KindNote: 6,
}

// Parse 는 마크다운 바이트를 IR Report 로 변환한다.
func Parse(data []byte) (*ir.Report, error) {
	text := strings.ReplaceAll(string(data), "\ufeff", "")
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = stripHTMLComments(text)

	fm, body := splitFrontmatter(text)
	rep := &ir.Report{
		Meta:      metaFromFrontmatter(fm),
		Footnotes: map[string]string{},
	}
	p := &parser{rep: rep, target: &rep.Body}

	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		i = p.handleLine(lines, i)
	}

	rep.Summary = p.buildSummary(fm)
	rep.Foreword = strings.TrimSpace(strings.Join(p.forewordBuf, " "))
	if len(rep.Footnotes) == 0 {
		rep.Footnotes = nil
	}
	return rep, nil
}

// handleLine 은 한 줄(필요 시 여러 줄: 표·코드펜스)을 처리하고 마지막 처리 인덱스를 반환한다.
func (p *parser) handleLine(lines []string, i int) int {
	raw := lines[i]
	s := strings.TrimRight(raw, " \t\r")
	stripped := strings.TrimSpace(s)

	if stripped == "" || reHRule.MatchString(stripped) {
		return i
	}

	// 각주 정의
	if m := reFnDef.FindStringSubmatch(stripped); m != nil {
		p.rep.Footnotes[m[1]] = stripInline(strings.TrimSpace(m[2]))
		return i
	}

	// 헤딩
	if m := reHeading.FindStringSubmatch(stripped); m != nil {
		level := len(m[1])
		label := trimNumber(stripInline(strings.TrimSpace(m[2])))
		if sp, ok := specialLabels[strings.ReplaceAll(label, " ", "")]; ok && level <= 2 {
			p.enterSpecial(sp)
			return i
		}
		// 특수섹션(요약 등) 내부의 ## 소제목은 블록 구분으로
		if p.mode == "summary" && level >= 2 {
			p.summaryBlocks = append(p.summaryBlocks, []string{})
			return i
		}
		p.addHeading(level, label)
		return i
	}

	// 표 캡션
	if !strings.HasPrefix(stripped, "|") {
		if m := reCaption.FindStringSubmatch(stripped); m != nil {
			p.pendingCaption = stripInline(strings.TrimSpace(m[1]))
			return i
		}
		if m := reCaption2.FindStringSubmatch(stripped); m != nil {
			p.pendingCaption = stripInline(strings.TrimSpace(m[1]))
			return i
		}
	}
	// 자료 출처
	if m := reSource.FindStringSubmatch(s); m != nil {
		p.pendingSource = stripInline(strings.TrimSpace(m[1]))
		return i
	}
	// 표
	if strings.HasPrefix(stripped, "|") {
		tbl, ni := parseTable(lines, i)
		if tbl != nil && p.acceptsBody() {
			tbl.Title = p.pendingCaption
			tbl.Source = p.pendingSource
			p.attach(&ir.Node{Kind: ir.KindTable, Payload: &ir.Payload{Table: tbl}})
		}
		p.pendingCaption, p.pendingSource = "", ""
		return ni
	}
	// 이미지
	if m := reImage.FindStringSubmatch(s); m != nil {
		if p.acceptsBody() {
			cap := stripInline(strings.TrimSpace(m[1]))
			if cap == "" {
				cap = p.pendingCaption
			}
			p.attach(&ir.Node{Kind: ir.KindFigure, Payload: &ir.Payload{
				Figure: &ir.Figure{Caption: cap, ImagePath: m[2]}}})
		}
		p.pendingCaption = ""
		return i
	}
	// 코드펜스 (```flow → 흐름도)
	if m := reFence.FindStringSubmatch(stripped); m != nil {
		lang := strings.ToLower(m[1])
		var block []string
		i++
		for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			block = append(block, lines[i])
			i++
		}
		if lang == "flow" && p.acceptsBody() {
			p.attach(&ir.Node{Kind: ir.KindFigure, Payload: &ir.Payload{
				Figure: &ir.Figure{Caption: p.pendingCaption, Flow: parseFlow(block)}}})
			p.pendingCaption = ""
		}
		return i
	}

	// 특수 섹션 본문
	switch p.mode {
	case "summary":
		if c := trimSymbol(stripInline(stripped)); c != "" {
			if len(p.summaryBlocks) == 0 {
				p.summaryBlocks = append(p.summaryBlocks, []string{})
			}
			n := len(p.summaryBlocks) - 1
			p.summaryBlocks[n] = append(p.summaryBlocks[n], c)
		}
		return i
	case "refs":
		p.rep.References = append(p.rep.References, trimSymbol(stripInline(stripped)))
		return i
	case "foreword":
		if c := trimSymbol(stripInline(stripped)); c != "" {
			p.forewordBuf = append(p.forewordBuf, c)
		}
		return i
	case "toc":
		return i // 목차는 자동 생성하므로 입력 무시
	}

	// 본문 불릿/문단
	kind := symbolKind(stripped)
	text := stripInline(stripped)
	if kind == "" {
		// 기호 없는 문단 → point
		p.attach(&ir.Node{Kind: ir.KindPoint, Text: text})
		return i
	}
	p.attach(&ir.Node{Kind: kind, Text: trimSymbol(text)})
	return i
}

func (p *parser) acceptsBody() bool {
	return p.mode == "" || p.mode == "appendix"
}

func (p *parser) enterSpecial(kind string) {
	p.stack = nil
	switch kind {
	case "appendix":
		p.mode = "appendix"
		p.target = &p.rep.Appendix
	case "summary", "refs", "foreword", "toc":
		p.mode = kind
	}
}

// addHeading 은 헤딩(장/절/세부/상세)을 트리에 추가한다.
func (p *parser) addHeading(level int, text string) {
	kind := map[int]string{1: ir.KindChapter, 2: ir.KindSection, 3: ir.KindSubsec, 4: ir.KindDetail}[level]
	if kind == "" {
		kind = ir.KindDetail
	}
	// 본문 장 시작 → 특수섹션 모드 종료
	if kind == ir.KindChapter && (p.mode == "summary" || p.mode == "refs" || p.mode == "foreword" || p.mode == "toc") {
		p.mode = ""
		p.target = &p.rep.Body
		p.stack = nil
	}
	node := &ir.Node{Kind: kind, Text: text}
	if kind == ir.KindChapter && p.mode != "appendix" {
		p.rep.Body = append(p.rep.Body, node)
		p.stack = []*stackItem{{depthOf[kind], node}}
		p.target = &p.rep.Body
		return
	}
	p.push(node, depthOf[kind])
}

// attach 는 노드를 스택 최상위(가장 가까운 상위)의 자식으로 붙인다.
func (p *parser) attach(node *ir.Node) {
	if len(p.stack) > 0 {
		top := p.stack[len(p.stack)-1].node
		top.Children = append(top.Children, node)
	} else {
		*p.target = append(*p.target, node)
	}
}

// push 는 계층 노드를 스택에 넣는다(depth 기준 상위로 되감기).
func (p *parser) push(node *ir.Node, depth int) {
	for len(p.stack) > 0 && p.stack[len(p.stack)-1].depth >= depth {
		p.stack = p.stack[:len(p.stack)-1]
	}
	if len(p.stack) > 0 {
		top := p.stack[len(p.stack)-1].node
		top.Children = append(top.Children, node)
	} else {
		*p.target = append(*p.target, node)
	}
	p.stack = append(p.stack, &stackItem{depth, node})
}

func (p *parser) buildSummary(fm map[string]string) ir.Summary {
	var flat []string
	for _, b := range p.summaryBlocks {
		if len(b) > 0 {
			flat = append(flat, strings.Join(b, " "))
		}
	}
	get := func(i int) string {
		if i < len(flat) {
			return flat[i]
		}
		return ""
	}
	return ir.Summary{
		Purpose:     firstNonEmpty(fm["summary_purpose"], get(0)),
		Contents:    firstNonEmpty(fm["summary_contents"], get(1)),
		Conclusion:  firstNonEmpty(fm["summary_conclusion"], get(2)),
		Utilization: firstNonEmpty(fm["summary_utilization"], get(3)),
	}
}

// symbolKind 는 줄 선두 기호로 본문 노드 종류를 판정한다.
func symbolKind(s string) string {
	switch {
	case strings.HasPrefix(s, "○ "), strings.HasPrefix(s, "● "), strings.HasPrefix(s, "◦ "):
		return ir.KindPoint
	case strings.HasPrefix(s, "– "), strings.HasPrefix(s, "— "), strings.HasPrefix(s, "- "):
		return ir.KindSub
	case strings.HasPrefix(s, "• "):
		return ir.KindSub
	case strings.HasPrefix(s, "※"):
		return ir.KindNote
	}
	return ""
}

func trimNumber(s string) string {
	return strings.TrimSpace(reNumPrefix.ReplaceAllString(strings.TrimSpace(s), ""))
}

func trimSymbol(s string) string {
	return strings.TrimSpace(reSymPrefix.ReplaceAllString(strings.TrimSpace(s), ""))
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
