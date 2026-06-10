package assemble

import (
	"regexp"
	"strconv"

	"github.com/beevik/etree"
	"hwpx-uri-gen/internal/hwpx"
	"hwpx-uri-gen/internal/rules"
)

var footnoteRe = regexp.MustCompile(`\[\^(\w+)\]`)

// stripFootnoteMarkers 는 [^N] 마커를 제거한다(각주 정의가 없을 때).
func stripFootnoteMarkers(s string) string {
	return footnoteRe.ReplaceAllString(s, "")
}

// applyText 는 단락에 텍스트를 넣되, [^N] 마커가 있고 각주 정의가 있으면
// 그 위치에 HWPX footNote 컨트롤 run 을 삽입한다.
func applyText(p *etree.Element, full string, ctx *buildCtx) {
	if applyFootnotes(p, full, ctx) {
		return
	}
	hwpx.SetParaText(p, stripFootnoteMarkers(full))
}

// applyFootnotes 는 [^N] 을 각주 run 으로 변환한다. 처리했으면 true.
func applyFootnotes(p *etree.Element, full string, ctx *buildCtx) bool {
	if len(ctx.footnotes) == 0 {
		return false
	}
	locs := footnoteRe.FindAllStringSubmatchIndex(full, -1)
	if len(locs) == 0 {
		return false
	}

	// 본문 run(첫 t 를 가진 run) 확보
	var baseRun *etree.Element
	for _, c := range p.ChildElements() {
		if c.Tag == "run" && findFirst(c, "t") != nil {
			baseRun = c
			break
		}
	}
	if baseRun == nil {
		return false
	}
	charPr := baseRun.SelectAttrValue("charPrIDRef", "0")

	// [text0][^k0][text1][^k1]…[tail] 로 분해
	type seg struct{ text, key string }
	var segs []seg
	last := 0
	for _, loc := range locs {
		segs = append(segs, seg{full[last:loc[0]], full[loc[2]:loc[3]]})
		last = loc[1]
	}
	tail := full[last:]

	// 첫 텍스트 조각을 baseRun 에
	setRunSingleText(baseRun, segs[0].text)
	insertAt := baseRun.Index() + 1
	add := func(el *etree.Element) {
		p.InsertChildAt(insertAt, el)
		insertAt++
	}

	for i, s := range segs {
		if content := ctx.footnotes[s.key]; content != "" {
			ctx.fnSeq++
			add(buildFootnoteRun(ctx.fnSeq, content, charPr, ctx.r))
		}
		next := tail
		if i+1 < len(segs) {
			next = segs[i+1].text
		}
		if next != "" {
			tr := baseRun.Copy()
			setRunSingleText(tr, next)
			add(tr)
		}
	}
	return true
}

// setRunSingleText 는 run 의 첫 t 에 text 를 넣고 나머지 t 를 비운다.
func setRunSingleText(run *etree.Element, text string) {
	var ts []*etree.Element
	if run.Tag == "t" {
		ts = append(ts, run)
	}
	hwpx.Walk(run, func(e *etree.Element) {
		if e.Tag == "t" {
			ts = append(ts, e)
		}
	})
	if len(ts) == 0 {
		t := run.CreateElement(prefixedHP(run, "t"))
		t.SetText(text)
		return
	}
	ts[0].Child = nil
	ts[0].SetText(text)
	for _, t := range ts[1:] {
		t.Child = nil
	}
}

// buildFootnoteRun 은 완성본 구조와 동일한 각주 컨트롤 run 을 만든다.
// instId(1900000000+N)·subList p id(2147483648+N)는 각주마다 고유해야
// 한글이 각주를 누락하지 않는다.
func buildFootnoteRun(num int, content, markerCharPr string, r *rules.Rules) *etree.Element {
	run := etree.NewElement("hp:run")
	run.CreateAttr("charPrIDRef", markerCharPr)
	ctrl := run.CreateElement("hp:ctrl")
	fn := ctrl.CreateElement("hp:footNote")
	fn.CreateAttr("number", strconv.Itoa(num))
	fn.CreateAttr("suffixChar", orDefault(r.Footnote.MarkerSuffix, "41"))
	fn.CreateAttr("instId", strconv.Itoa(1900000000+num))

	sl := fn.CreateElement("hp:subList")
	for _, kv := range [][2]string{
		{"id", ""}, {"textDirection", "HORIZONTAL"}, {"lineWrap", "BREAK"},
		{"vertAlign", "TOP"}, {"linkListIDRef", "0"}, {"linkListNextIDRef", "0"},
		{"textWidth", "0"}, {"textHeight", "0"}, {"hasTextRef", "0"}, {"hasNumRef", "0"},
	} {
		sl.CreateAttr(kv[0], kv[1])
	}

	fp := sl.CreateElement("hp:p")
	fp.CreateAttr("id", strconv.Itoa(2147483648+num))
	fp.CreateAttr("paraPrIDRef", orDefault(r.Footnote.ContentPara, "4"))
	fp.CreateAttr("styleIDRef", orDefault(r.Footnote.ContentStyle, "8"))
	fp.CreateAttr("pageBreak", "0")
	fp.CreateAttr("columnBreak", "0")
	fp.CreateAttr("merged", "0")

	r2 := fp.CreateElement("hp:run")
	r2.CreateAttr("charPrIDRef", orDefault(r.Footnote.ContentCharPr, "0"))
	c2 := r2.CreateElement("hp:ctrl")
	an := c2.CreateElement("hp:autoNum")
	an.CreateAttr("num", strconv.Itoa(num))
	an.CreateAttr("numType", "FOOTNOTE")
	anf := an.CreateElement("hp:autoNumFormat")
	anf.CreateAttr("type", "DIGIT")
	anf.CreateAttr("userChar", "")
	anf.CreateAttr("prefixChar", "")
	anf.CreateAttr("suffixChar", orDefault(r.Footnote.NumSuffix, ")"))
	anf.CreateAttr("supscript", "0")
	t := r2.CreateElement("hp:t")
	t.SetText(" " + content)
	return run
}

func prefixedHP(sibling *etree.Element, tag string) string {
	if sibling.Space == "" {
		return tag
	}
	return sibling.Space + ":" + tag
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
