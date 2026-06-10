package assemble

import (
	"fmt"

	"github.com/beevik/etree"
	"hwpx-uri-gen/internal/hwpx"
	"hwpx-uri-gen/internal/rules"
)

// anchors 는 양식 템플릿에서 추출한 계층별 참조 단락(복제본)이다.
// 조립기는 이 단락을 ClonePara 해 서식을 통째로 물려받고 텍스트만 교체한다.
type anchors struct {
	chapterCover  *etree.Element // 간지 (secPr 포함 — 섹션/페이지 리셋)
	chapterTitle  *etree.Element // 장 제목 (run0=번호, run1=제목)
	chapterToc    *etree.Element // 간지 미니목차 표
	sectionFirst  *etree.Element // 첫 소절
	sectionRest   *etree.Element // 둘째 이후 소절
	subsec        *etree.Element // 세부/상세 (같은 서식)
	point         *etree.Element // 주요 항목 ○
	sub           *etree.Element // 세부 설명 –
	note          *etree.Element // 비고 ※ (없을 수 있음 → point fallback)
	tableCaption  *etree.Element // [표 N-M] 캡션
	tableGrid     *etree.Element // 데이터표
	source        *etree.Element // 자료 출처
	figureArea    *etree.Element // 그림 영역
	figureCaption *etree.Element // [그림 N-M] 캡션
}

// findByStylePara 는 sec 직계 단락 중 styleIDRef·paraPrIDRef 가 일치하는 첫 단락을 찾는다.
func findByStylePara(sec *etree.Element, style, para string) *etree.Element {
	for _, p := range hwpx.SecParas(sec) {
		if p.SelectAttrValue("styleIDRef", "") == style &&
			p.SelectAttrValue("paraPrIDRef", "") == para {
			return p
		}
	}
	return nil
}

// findByStyleParaTbl 는 style·para 일치하면서 표(tbl)를 가진 첫 단락을 찾는다.
func findByStyleParaTbl(sec *etree.Element, style, para string) *etree.Element {
	for _, p := range hwpx.SecParas(sec) {
		if p.SelectAttrValue("styleIDRef", "") != style ||
			p.SelectAttrValue("paraPrIDRef", "") != para {
			continue
		}
		has := false
		hwpx.Walk(p, func(e *etree.Element) {
			if e.Tag == "tbl" {
				has = true
			}
		})
		if has {
			return p
		}
	}
	return nil
}

// collectAnchors 는 템플릿 section1 에서 모든 계층 앵커를 추출·복제한다.
// 필수 앵커(간지·장제목·소절·주요)가 없으면 에러.
func collectAnchors(sec *etree.Element, r *rules.Rules) (*anchors, error) {
	clone := func(e *etree.Element) *etree.Element {
		if e == nil {
			return nil
		}
		return hwpx.ClonePara(e)
	}

	a := &anchors{
		chapterCover:  clone(findByStylePara(sec, r.Levels.Chapter.Cover.Style, r.Levels.Chapter.Cover.Para)),
		chapterTitle:  clone(findByStylePara(sec, r.Levels.Chapter.Title.Style, r.Levels.Chapter.Title.Para)),
		chapterToc:    clone(findByStyleParaTbl(sec, r.Levels.Chapter.Toc.Style, r.Levels.Chapter.Toc.Para)),
		sectionFirst:  clone(findByStylePara(sec, r.Levels.Section.Style, r.Levels.Section.ParaFirst)),
		sectionRest:   clone(findByStylePara(sec, r.Levels.Section.Style, r.Levels.Section.ParaRest)),
		subsec:        clone(findByStylePara(sec, r.Levels.Subsec.Style, r.Levels.Subsec.Para)),
		point:         clone(findByStylePara(sec, r.Levels.Point.Style, r.Levels.Point.Para)),
		sub:           clone(findByStylePara(sec, r.Levels.Sub.Style, r.Levels.Sub.Para)),
		note:          clone(findByStylePara(sec, r.Levels.Note.Style, r.Levels.Note.Para)),
		tableCaption:  clone(findByStylePara(sec, r.Table.Caption.Style, r.Table.Caption.Para)),
		tableGrid:     clone(findByStyleParaTbl(sec, r.Table.Grid.Style, r.Table.Grid.Para)),
		source:        clone(findByStylePara(sec, r.Source.Style, r.Source.Para)),
		figureArea:    clone(findByStylePara(sec, r.Figure.Area.Style, r.Figure.Area.Para)),
		figureCaption: clone(findByStylePara(sec, r.Figure.Caption.Style, r.Figure.Caption.Para)),
	}

	// 필수 앵커 검증
	missing := []string{}
	if a.chapterCover == nil {
		missing = append(missing, "간지(chapter.cover)")
	}
	if a.chapterTitle == nil {
		missing = append(missing, "장제목(chapter.title)")
	}
	if a.sectionFirst == nil {
		missing = append(missing, "소절(section)")
	}
	if a.point == nil {
		missing = append(missing, "주요항목(point)")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("템플릿에서 필수 앵커를 찾지 못함: %v", missing)
	}

	// 선택 앵커 fallback
	if a.sectionRest == nil {
		a.sectionRest = clone(a.sectionFirst)
	}
	if a.sub == nil {
		a.sub = clone(a.point)
	}
	if a.note == nil {
		a.note = clone(a.point)
	}
	if a.subsec == nil {
		a.subsec = clone(a.point)
	}
	if a.source == nil {
		a.source = clone(a.point)
	}
	return a, nil
}
