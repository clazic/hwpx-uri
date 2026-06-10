// Package profile 은 양식 hwpx 를 분석해 "양식 적응형" 변환의 토대를 만든다.
//
// 두 가지를 제공한다:
//  1. 스타일 이름 → ID 해석 (header.xml) — 스타일 ID 를 코드에 하드코딩하지 않으므로
//     양식이 ID 를 재배치해도 이름이 같으면 따라간다. 주로 검증·필수스타일 확인용.
//  2. 앵커 단락 탐색 (section XML) — 계층별 참조 단락을 스타일·기호·텍스트 패턴으로 찾는다.
//     조립기는 이 단락을 복제(ClonePara)해 서식을 통째로 물려받고 텍스트만 교체한다.
//     placeholder 텍스트("[챕터 제목]" 등)가 유지되면 양식이 개정돼도 따라간다.
package profile

import (
	"strings"

	"github.com/beevik/etree"
	"hwpx-uri-gen/internal/hwpx"
)

// StyleEntry 는 한 스타일의 참조 ID 들이다.
type StyleEntry struct {
	ID          string
	ParaPrIDRef string
	CharPrIDRef string
}

// Profile 은 양식에서 추출한 스타일 사전이다.
type Profile struct {
	Styles map[string]StyleEntry // 원본 이름 + strip 별칭 둘 다 키로 등록
}

// 변환에 필수인 스타일 이름(strip 기준). 하나라도 없으면 양식 호환성 의심.
var RequiredStyleNames = []string{
	"바탕글", "□", "○", "-", "※",
	"제1장", "1 2 3", "1) 2) 3)", "(1)",
	"간지", "표제목", "표머리", "표내용",
	"자료", "참고문헌-내용", "부록",
}

// Extract 는 header.xml(etree 문서)에서 스타일 사전을 만든다.
func Extract(headerDoc *etree.Document) *Profile {
	p := &Profile{Styles: make(map[string]StyleEntry)}
	root := headerDoc.Root()
	if root == nil {
		return p
	}
	hwpx.Walk(root, func(e *etree.Element) {
		if e.Tag != "style" {
			return
		}
		name := e.SelectAttrValue("name", "")
		if name == "" {
			return
		}
		entry := StyleEntry{
			ID:          e.SelectAttrValue("id", ""),
			ParaPrIDRef: e.SelectAttrValue("paraPrIDRef", ""),
			CharPrIDRef: e.SelectAttrValue("charPrIDRef", ""),
		}
		p.Styles[name] = entry
		if s := strings.TrimSpace(name); s != "" && s != name {
			if _, ok := p.Styles[s]; !ok {
				p.Styles[s] = entry
			}
		}
	})
	return p
}

// StyleID 는 이름(원본 또는 strip)으로 styleID 를 반환한다. 없으면 "".
func (p *Profile) StyleID(name string) string {
	if e, ok := p.Styles[name]; ok && e.ID != "" {
		return e.ID
	}
	if e, ok := p.Styles[strings.TrimSpace(name)]; ok {
		return e.ID
	}
	return ""
}

// ParaPrID 는 이름으로 paraPrIDRef 를 반환한다. 없으면 "".
func (p *Profile) ParaPrID(name string) string {
	if e, ok := p.Styles[name]; ok {
		return e.ParaPrIDRef
	}
	if e, ok := p.Styles[strings.TrimSpace(name)]; ok {
		return e.ParaPrIDRef
	}
	return ""
}

// MissingRequired 는 필수 스타일 중 양식에 없는 이름 목록을 반환한다.
func (p *Profile) MissingRequired() []string {
	var missing []string
	for _, n := range RequiredStyleNames {
		if _, ok := p.Styles[n]; !ok {
			missing = append(missing, n)
		}
	}
	return missing
}

// AnchorRule 은 section 에서 참조 단락을 찾는 조건이다(주어진 항목만 검사).
type AnchorRule struct {
	StyleID string // 단락 styleIDRef 가 이 값과 일치
	Sym     string // 단락 텍스트(trim)가 이 기호로 시작
	Text    string // 단락 텍스트에 이 문자열 포함
	HasTbl  bool   // 단락 안에 표(tbl)가 있어야 함
}

// FindAnchor 는 sec 직계 단락 중 rule 을 모두 만족하는 첫 단락을 반환한다(없으면 nil).
func FindAnchor(sec *etree.Element, rule AnchorRule) *etree.Element {
	for _, p := range hwpx.SecParas(sec) {
		if rule.StyleID != "" && p.SelectAttrValue("styleIDRef", "") != rule.StyleID {
			continue
		}
		text := hwpx.ParaText(p)
		trimmed := strings.TrimSpace(text)
		if rule.Sym != "" && !strings.HasPrefix(trimmed, rule.Sym) {
			continue
		}
		if rule.Text != "" && !strings.Contains(text, rule.Text) {
			continue
		}
		if rule.HasTbl && !hasTbl(p) {
			continue
		}
		return p
	}
	return nil
}

func hasTbl(p *etree.Element) bool {
	found := false
	hwpx.Walk(p, func(e *etree.Element) {
		if e.Tag == "tbl" {
			found = true
		}
	})
	return found
}
