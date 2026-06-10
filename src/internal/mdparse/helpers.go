package mdparse

import (
	"regexp"
	"strings"

	"hwpx-uri-gen/internal/ir"
)

var (
	reComment   = regexp.MustCompile(`(?s)<!--.*?-->`)
	reMdImage   = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	reMdLink    = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	reBold      = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reBold2     = regexp.MustCompile(`__([^_]+)__`)
	reItalic    = regexp.MustCompile(`(?:^|[^*\w])\*([^*\n]+)\*(?:[^*\w]|$)`)
	reCode      = regexp.MustCompile("`([^`]+)`")
	reTableSep  = regexp.MustCompile(`^:?-{2,}:?$`)
	reFrontYAML = regexp.MustCompile(`(?s)^\x{feff}?---\s*\n(.*?)\n---\s*\n?(.*)$`)
)

func stripHTMLComments(s string) string {
	return reComment.ReplaceAllString(s, "")
}

// splitFrontmatter 는 YAML frontmatter(---...---)를 map 으로, 나머지를 본문으로 분리한다.
func splitFrontmatter(text string) (map[string]string, string) {
	m := reFrontYAML.FindStringSubmatch(text)
	if m == nil {
		return map[string]string{}, text
	}
	fm := map[string]string{}
	for _, line := range strings.Split(m[1], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, ":") {
			continue
		}
		k, v, _ := strings.Cut(line, ":")
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		fm[strings.TrimSpace(k)] = v
	}
	return fm, m[2]
}

// metaFromFrontmatter 는 frontmatter map 을 표지 메타로 매핑한다(한/영 키 모두).
func metaFromFrontmatter(fm map[string]string) ir.Meta {
	pick := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := fm[k]; ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	return ir.Meta{
		Title:    pick("title", "제목", "보고서제목"),
		Series:   pick("series", "시리즈", "과제번호"),
		Lab:      pick("lab", "연구실", "연구실명", "department"),
		Author:   pick("author", "저자", "연구자", "연구책임"),
		PubDate:  pick("date", "pub_date", "발간월", "발간년월", "발간일"),
		Director: pick("director", "원장", "원장명"),
	}
}

// stripInline 은 인라인 마크업(**굵게** *기울임* `코드` [링크] ![이미지])을 제거한다.
// 각주 마커 [^N] 은 보존한다(조립기가 각주로 변환).
func stripInline(s string) string {
	if s == "" {
		return s
	}
	s = reMdImage.ReplaceAllString(s, "$1")
	// [^N] 보호: 링크 정규식이 건드리지 않도록 먼저 치환
	s = protectFootnotes(s, func(t string) string {
		t = reMdLink.ReplaceAllString(t, "$1")
		return t
	})
	s = reBold.ReplaceAllString(s, "$1")
	s = reBold2.ReplaceAllString(s, "$1")
	s = reCode.ReplaceAllString(s, "$1")
	return strings.TrimSpace(s)
}

// protectFootnotes 는 [^N] 을 보호한 채 fn 을 적용한다.
func protectFootnotes(s string, fn func(string) string) string {
	// [^N] 은 reMdLink( [text](url) )와 형태가 달라 실제로는 충돌하지 않지만,
	// 안전을 위해 그대로 통과시킨다.
	return fn(s)
}

// parseTable 은 연속된 | 행들을 표로 파싱한다. 구분선(---)은 제거한다.
func parseTable(lines []string, i int) (*ir.Table, int) {
	var rows [][]string
	for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
		line := strings.TrimSpace(lines[i])
		line = strings.Trim(line, "|")
		var cells []string
		for _, c := range strings.Split(line, "|") {
			cells = append(cells, stripInline(strings.TrimSpace(c)))
		}
		rows = append(rows, cells)
		i++
	}
	// 구분선 행 제거
	var clean [][]string
	for _, r := range rows {
		sep := true
		for _, c := range r {
			cc := c
			if cc == "" {
				cc = "-"
			}
			if !reTableSep.MatchString(cc) {
				sep = false
				break
			}
		}
		if !sep {
			clean = append(clean, r)
		}
	}
	if len(clean) == 0 {
		return nil, i
	}
	return &ir.Table{Headers: clean[0], Rows: clean[1:]}, i - 1
}

// parseFlow 는 flow 코드블록을 흐름도 스펙으로 파싱한다.
// '|' 가 있는 첫 줄은 상단 병렬 박스, 나머지는 세로 단계.
func parseFlow(block []string) *ir.Flow {
	f := &ir.Flow{}
	for _, ln := range block {
		s := strings.TrimSpace(ln)
		if s == "" {
			continue
		}
		if strings.Contains(s, "|") && len(f.ParallelTop) == 0 {
			for _, x := range strings.Split(s, "|") {
				if x = strings.TrimSpace(x); x != "" {
					f.ParallelTop = append(f.ParallelTop, x)
				}
			}
		} else {
			f.Steps = append(f.Steps, s)
		}
	}
	return f
}
