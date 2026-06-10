package assemble

import (
	"fmt"

	"github.com/beevik/etree"
	"hwpx-uri-gen/internal/hwpx"
	"hwpx-uri-gen/internal/ir"
	"hwpx-uri-gen/internal/rules"
)

// buildCtx 는 조립 중 공유되는 컨텍스트다(규칙·앵커·패키지·각주).
type buildCtx struct {
	r         *rules.Rules
	a         *anchors
	pkg       *hwpx.Package     // 이미지 동적 삽입용
	footnotes map[string]string // 각주 정의 {"1": 내용}
	fnSeq     int               // 각주 일련번호(전역 증가)
}

// counters 는 장별 번호 카운터다. 장이 바뀌면 하위가 리셋된다.
type counters struct {
	chapter, section, subsec, detail, table, figure int
}

// buildBodyParas 는 IR 본문(+참고문헌·부록)을 section1 단락 리스트로 만든다.
func buildBodyParas(rep *ir.Report, ctx *buildCtx) []*etree.Element {
	r, a := ctx.r, ctx.a
	var out []*etree.Element

	for ci, ch := range rep.Body {
		cn := ci + 1
		title := rules.TrimNumber(ch.Text)

		// 간지 (secPr 포함 → 새 장/페이지 리셋). run1=번호, run2의 마지막 t=제목
		cover := hwpx.ClonePara(a.chapterCover)
		hwpx.SetRunText(cover, 1, r.RomanNum(cn))
		hwpx.SetRunTextLast(cover, 2, title)
		// 쪽번호: 1장 간지의 newNum(=1)만 남기면 본문 전체가 1부터 연속 번호가 된다.
		// (앞부속은 masterpage0/1 이라 번호 없음 → 1장 첫 쪽이 1쪽)
		if cn > 1 && !r.Layout.PageNumRestartPerChapter {
			hwpx.RemovePageNumReset(cover)
		}
		out = append(out, cover)

		// 간지 미니목차 표: 그 장의 소절 목록
		if a.chapterToc != nil {
			titles := chapterSectionTitles(ch, r)
			if toc := buildChapterToc(a.chapterToc, titles); toc != nil {
				out = append(out, toc)
			}
		}

		// 장제목. run0=번호("Ⅰ. "), run1=제목
		t := hwpx.ClonePara(a.chapterTitle)
		hwpx.SetRunText(t, 0, r.Format(r.Levels.Chapter.Number, cn))
		hwpx.SetRunText(t, 1, title)
		out = append(out, t)

		ctr := &counters{chapter: cn}
		for _, node := range ch.Children {
			emitNode(node, ctx, ctr, &out)
		}
	}

	// 참고문헌
	if len(rep.References) > 0 {
		out = append(out, buildReferenceTitle(ctx))
		for _, ref := range rep.References {
			out = append(out, buildReferenceItem(ref, ctx))
		}
	}

	// 부록
	if len(rep.Appendix) > 0 {
		out = append(out, buildAppendixTitle(ctx))
		ctr := &counters{}
		for _, node := range rep.Appendix {
			emitNode(node, ctx, ctr, &out)
		}
	}

	return out
}

// emitNode 는 한 노드를 단락으로 변환해 out 에 추가한다(자식 재귀).
func emitNode(n *ir.Node, ctx *buildCtx, ctr *counters, out *[]*etree.Element) {
	r, a := ctx.r, ctx.a
	switch n.Kind {
	case ir.KindSection:
		ctr.section++
		ctr.subsec, ctr.detail = 0, 0
		ref := a.sectionRest
		if ctr.section == 1 {
			ref = a.sectionFirst
		}
		p := hwpx.ClonePara(ref)
		applyText(p, r.Format(r.Levels.Section.Number, ctr.section)+rules.TrimNumber(n.Text), ctx)
		*out = append(*out, p)

	case ir.KindSubsec:
		ctr.subsec++
		ctr.detail = 0
		if r.Layout.BlankBetweenSubsec && ctr.subsec > 1 {
			*out = append(*out, blankPara(a))
		}
		p := hwpx.ClonePara(a.subsec)
		applyText(p, r.Format(r.Levels.Subsec.Number, ctr.subsec)+rules.TrimNumber(n.Text), ctx)
		*out = append(*out, p)

	case ir.KindDetail:
		ctr.detail++
		if r.Layout.BlankBetweenDetail && ctr.detail > 1 {
			*out = append(*out, blankPara(a))
		}
		p := hwpx.ClonePara(a.subsec)
		applyText(p, r.Format(r.Levels.Detail.Number, ctr.detail)+rules.TrimNumber(n.Text), ctx)
		*out = append(*out, p)

	case ir.KindPoint:
		p := hwpx.ClonePara(a.point)
		applyText(p, r.Levels.Point.Symbol+n.Text, ctx)
		*out = append(*out, p)

	case ir.KindSub:
		p := hwpx.ClonePara(a.sub)
		applyText(p, r.Levels.Sub.Symbol+n.Text, ctx)
		*out = append(*out, p)

	case ir.KindNote:
		p := hwpx.ClonePara(a.note)
		applyText(p, r.Levels.Note.Symbol+n.Text, ctx)
		*out = append(*out, p)

	case ir.KindPara:
		p := hwpx.ClonePara(a.point)
		applyText(p, n.Text, ctx)
		*out = append(*out, p)

	case ir.KindTable:
		ctr.table++
		emitTable(n, ctr, ctx, out)

	case ir.KindFigure:
		ctr.figure++
		emitFigure(n, ctr, ctx, out)
	}

	for _, c := range n.Children {
		emitNode(c, ctx, ctr, out)
	}
}

// blankPara 는 빈 한 줄(항목 간 시각적 구분)을 만든다.
func blankPara(a *anchors) *etree.Element {
	p := hwpx.ClonePara(a.point)
	hwpx.SetParaText(p, "")
	return p
}

// chapterSectionTitles 는 한 장의 소절 제목들(번호 포함)을 모은다(미니목차용).
func chapterSectionTitles(ch *ir.Node, r *rules.Rules) []string {
	var titles []string
	n := 0
	for _, c := range ch.Children {
		if c.Kind == ir.KindSection {
			n++
			titles = append(titles, r.Format(r.Levels.Section.Number, n)+rules.TrimNumber(c.Text))
		}
	}
	return titles
}

func buildReferenceTitle(ctx *buildCtx) *etree.Element {
	p := hwpx.ClonePara(ctx.a.chapterTitle)
	hwpx.SetRunText(p, 0, "")
	hwpx.SetRunText(p, 1, "참고문헌")
	return p
}

func buildReferenceItem(text string, ctx *buildCtx) *etree.Element {
	p := hwpx.ClonePara(ctx.a.point)
	hwpx.SetParaText(p, ctx.r.References.Item.Symbol+rules.TrimNumber(text))
	return p
}

func buildAppendixTitle(ctx *buildCtx) *etree.Element {
	p := hwpx.ClonePara(ctx.a.chapterTitle)
	hwpx.SetRunText(p, 0, "")
	hwpx.SetRunText(p, 1, "부록")
	return p
}

// fmtCaption 은 "[표 N-M]  제목" / "[그림 N-M]  제목" 캡션을 만든다.
func fmtCaption(format string, chap, n int, title string) string {
	return fmt.Sprintf(format, chap, n) + title
}
