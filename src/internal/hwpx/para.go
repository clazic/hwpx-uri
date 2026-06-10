package hwpx

import "github.com/beevik/etree"

// etree 에서 Element.Tag 는 이미 local name(prefix 제외)이고 Element.Space 가 prefix 다.
// 따라서 태그 비교는 e.Tag 로 충분하다.

// Walk 는 root 의 모든 후손 요소에 fn 을 적용한다(전위 순회, root 자신 제외).
func Walk(root *etree.Element, fn func(*etree.Element)) {
	for _, c := range root.ChildElements() {
		fn(c)
		Walk(c, fn)
	}
}

// FindSec 은 문서에서 <hs:sec>(섹션 루트) 요소를 찾는다. 없으면 문서 루트를 반환한다.
func FindSec(doc *etree.Document) *etree.Element {
	root := doc.Root()
	if root == nil {
		return nil
	}
	if root.Tag == "sec" {
		return root
	}
	var found *etree.Element
	Walk(root, func(e *etree.Element) {
		if found == nil && e.Tag == "sec" {
			found = e
		}
	})
	if found != nil {
		return found
	}
	return root
}

// SecParas 는 sec 직계 자식 중 단락(<hp:p>)들을 순서대로 반환한다.
func SecParas(sec *etree.Element) []*etree.Element {
	var ps []*etree.Element
	for _, c := range sec.ChildElements() {
		if c.Tag == "p" {
			ps = append(ps, c)
		}
	}
	return ps
}

// findAllT 는 root 내 모든 <hp:t> 요소를 반환한다(root 자신 포함).
func findAllT(root *etree.Element) []*etree.Element {
	var ts []*etree.Element
	if root.Tag == "t" {
		ts = append(ts, root)
	}
	Walk(root, func(e *etree.Element) {
		if e.Tag == "t" {
			ts = append(ts, e)
		}
	})
	return ts
}

// runChildren 은 단락 직계의 <hp:run> 들을 반환한다.
func runChildren(p *etree.Element) []*etree.Element {
	var rs []*etree.Element
	for _, c := range p.ChildElements() {
		if c.Tag == "run" {
			rs = append(rs, c)
		}
	}
	return rs
}

// RemoveLineSegArray 는 root 이하 모든 linesegarray(줄바꿈 좌표 캐시)를 제거한다.
// 텍스트를 바꾼 단락에서 이 캐시가 남으면 macOS 한글 뷰어가 줄바꿈을 잘못 그린다.
func RemoveLineSegArray(root *etree.Element) {
	var targets []*etree.Element
	Walk(root, func(e *etree.Element) {
		if e.Tag == "linesegarray" {
			targets = append(targets, e)
		}
	})
	for _, t := range targets {
		if parent := t.Parent(); parent != nil {
			parent.RemoveChild(t)
		}
	}
}

// ClonePara 는 참조 단락을 깊은 복사하고 linesegarray 를 제거한다.
// 서식·네임스페이스·run 구조를 그대로 보존한 빈 틀을 만든다.
func ClonePara(ref *etree.Element) *etree.Element {
	p := ref.Copy()
	RemoveLineSegArray(p)
	return p
}

// SetParaText 는 단락의 첫 번째 <hp:t> 에 text 를 넣고 나머지 t 의 내용을 비운다.
// 각 t 의 기존 내용(자식 tab·CharData 포함)은 완전히 제거된다.
// 기호가 양식 단락에 박혀 있는 계층(○·-·※ 등)에서는 text 에 기호를 포함하지 않는다.
func SetParaText(p *etree.Element, text string) {
	ts := findAllT(p)
	if len(ts) == 0 {
		return
	}
	ts[0].Child = nil
	ts[0].SetText(text)
	for _, t := range ts[1:] {
		t.Child = nil
	}
}

// SetRunText 는 단락의 runIdx 번째 run 안 첫 <hp:t> 에 text 를 넣는다.
// 간지/장제목처럼 run 이 번호·제목으로 나뉜 단락에서 run 별로 교체할 때 쓴다.
// 해당 run 에 t 가 없으면 새로 만든다(run 의 prefix 를 따른다).
func SetRunText(p *etree.Element, runIdx int, text string) {
	rs := runChildren(p)
	if runIdx < 0 || runIdx >= len(rs) {
		return
	}
	run := rs[runIdx]
	ts := findAllT(run)
	if len(ts) == 0 {
		t := run.CreateElement(prefixed(run.Space, "t"))
		t.SetText(text)
		return
	}
	ts[0].Child = nil
	ts[0].SetText(text)
	for _, t := range ts[1:] {
		t.Child = nil
	}
}

// SetRunTextLast 는 runIdx 번째 run 의 "마지막" <hp:t> 에 text 를 넣고 앞 t 들은 그대로 둔다.
// 양식 간지처럼 run 안 첫 t 는 빈 채로 두고 마지막 t 에 제목이 들어가는 구조를 보존한다.
func SetRunTextLast(p *etree.Element, runIdx int, text string) {
	rs := runChildren(p)
	if runIdx < 0 || runIdx >= len(rs) {
		return
	}
	ts := findAllT(rs[runIdx])
	if len(ts) == 0 {
		t := rs[runIdx].CreateElement(prefixed(rs[runIdx].Space, "t"))
		t.SetText(text)
		return
	}
	last := ts[len(ts)-1]
	last.Child = nil
	last.SetText(text)
}

// RemovePageNumReset 은 단락 안의 쪽번호 리셋 컨트롤(ctrl>newNum numType=PAGE)을 제거한다.
// 완성본 양식은 장 간지마다 newNum 으로 쪽번호를 1로 되돌리는데, 연속 번호가 필요할 때
// 2장 이후 간지 복제본에서 이 컨트롤만 걷어낸다(secPr 등 나머지는 보존).
func RemovePageNumReset(p *etree.Element) {
	var targets []*etree.Element
	Walk(p, func(e *etree.Element) {
		if e.Tag == "newNum" && e.SelectAttrValue("numType", "") == "PAGE" {
			targets = append(targets, e)
		}
	})
	for _, nn := range targets {
		parent := nn.Parent()
		if parent == nil {
			continue
		}
		parent.RemoveChild(nn)
		// 빈 껍데기가 된 ctrl 래퍼도 제거
		if parent.Tag == "ctrl" && len(parent.ChildElements()) == 0 {
			if gp := parent.Parent(); gp != nil {
				gp.RemoveChild(parent)
			}
		}
	}
}

// SetStyleRefs 는 단락의 styleIDRef / paraPrIDRef 속성을 갱신한다(빈 값은 건너뜀).
func SetStyleRefs(p *etree.Element, styleID, paraPrID string) {
	if styleID != "" {
		p.CreateAttr("styleIDRef", styleID)
	}
	if paraPrID != "" {
		p.CreateAttr("paraPrIDRef", paraPrID)
	}
}

// ParaText 는 단락 안 모든 t 텍스트를 이어붙여 반환한다(앵커 탐색·디버깅용).
func ParaText(p *etree.Element) string {
	var s string
	for _, t := range findAllT(p) {
		s += t.Text()
	}
	return s
}

// prefixed 는 space 가 있으면 "space:tag", 없으면 "tag" 를 만든다.
func prefixed(space, tag string) string {
	if space == "" {
		return tag
	}
	return space + ":" + tag
}
