package assemble

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"hwpx-uri-gen/internal/hwpx"
	"hwpx-uri-gen/internal/imagegen"
	"hwpx-uri-gen/internal/ir"
)

// emitTable 은 표 세트(캡션 + 데이터표 + 자료)를 만든다.
func emitTable(n *ir.Node, ctr *counters, ctx *buildCtx, out *[]*etree.Element) {
	r, a := ctx.r, ctx.a
	t := n.Payload.Table

	if a.tableCaption != nil {
		cap := hwpx.ClonePara(a.tableCaption)
		hwpx.SetParaText(cap, fmtCaption(r.Table.CaptionFmt, ctr.chapter, ctr.table, t.Title))
		*out = append(*out, cap)
	}
	if a.tableGrid != nil && (len(t.Headers) > 0 || len(t.Rows) > 0) {
		if grid := buildTableGrid(a.tableGrid, t, ctx); grid != nil {
			*out = append(*out, grid)
		}
	}
	if t.Source != "" && a.source != nil {
		s := hwpx.ClonePara(a.source)
		hwpx.SetParaText(s, r.Source.Prefix+t.Source)
		*out = append(*out, s)
	}
}

// emitFigure 는 그림 세트(이미지/흐름도 영역 + 캡션)를 만든다.
func emitFigure(n *ir.Node, ctr *counters, ctx *buildCtx, out *[]*etree.Element) {
	r, a := ctx.r, ctx.a
	f := n.Payload.Figure

	var area *etree.Element
	switch {
	case f.Flow != nil && ctx.pkg != nil && a.figureArea != nil:
		area = buildFlowImage(f, ctx)
	case f.Chart != nil && ctx.pkg != nil && a.figureArea != nil:
		area = buildChartImage(f, ctx)
	case f.ImagePath != "" && ctx.pkg != nil && a.figureArea != nil:
		area = buildExternalImage(f, ctx)
	}

	// 이미지 생성 실패 또는 미지정 → placeholder
	if area == nil && a.figureArea != nil {
		area = hwpx.ClonePara(a.figureArea)
		if findFirst(area, "container") == nil { // 텍스트형 placeholder 단락일 때만
			label := "[그림 영역]"
			if f.Caption != "" {
				label = "[그림 영역 — " + f.Caption + "]"
			}
			hwpx.SetParaText(area, label)
		}
	}
	if area != nil {
		*out = append(*out, area)
	}

	if a.figureCaption != nil {
		cap := hwpx.ClonePara(a.figureCaption)
		hwpx.SetParaText(cap, fmtCaption(r.Figure.CaptionFmt, ctr.chapter, ctr.figure, f.Caption))
		*out = append(*out, cap)
	}
}

// buildFlowImage 는 flow 스펙으로 흐름도 PNG 를 생성·등록하고 그림 단락을 만든다.
func buildFlowImage(f *ir.Figure, ctx *buildCtx) *etree.Element {
	tmp, err := os.CreateTemp("", "flow-*.png")
	if err != nil {
		return nil
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	w, h, err := imagegen.MakeFlowchart(tmp.Name(), f.Flow.Title, f.Flow.ParallelTop, f.Flow.Steps)
	if err != nil {
		fmt.Printf("[figure] 흐름도 생성 실패: %v\n", err)
		return nil
	}
	return placeImage(ctx, tmp.Name(), w, h)
}

// buildChartImage 는 chart 스펙으로 막대그래프 PNG 를 생성·등록하고 그림 단락을 만든다.
func buildChartImage(f *ir.Figure, ctx *buildCtx) *etree.Element {
	tmp, err := os.CreateTemp("", "chart-*.png")
	if err != nil {
		return nil
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	w, h, err := imagegen.MakeBarChart(tmp.Name(), f.Chart.Title, f.Chart.Labels, f.Chart.Series)
	if err != nil {
		fmt.Printf("[figure] 차트 생성 실패: %v\n", err)
		return nil
	}
	return placeImage(ctx, tmp.Name(), w, h)
}

// buildExternalImage 는 외부 이미지 파일을 등록하고 그림 단락을 만든다.
func buildExternalImage(f *ir.Figure, ctx *buildCtx) *etree.Element {
	st, err := os.Stat(f.ImagePath)
	if err != nil || st.IsDir() {
		fmt.Printf("[figure] 이미지 파일 없음: %s\n", f.ImagePath)
		return nil
	}
	w, h := imagegen.PNGSize(f.ImagePath) // png 외 형식은 0,0 → 기본 비율
	return placeImage(ctx, f.ImagePath, w, h)
}

// placeImage 는 이미지 파일을 패키지에 등록하고 container 앵커에 삽입한 단락을 반환한다.
func placeImage(ctx *buildCtx, path string, w, h int) *etree.Element {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	ext := strings.TrimPrefix(strings.ToLower(pathExt(path)), ".")
	imgID, err := ctx.pkg.AddImage(data, ext)
	if err != nil {
		fmt.Printf("[figure] 이미지 등록 실패: %v\n", err)
		return nil
	}
	area := hwpx.ClonePara(ctx.a.figureArea)
	// 본문 폭 기준(완성본 그림 너비 42000 HWPUNIT), 비율 유지
	const baseW = 42000
	hwpW, hwpH := baseW, 31542
	if w > 0 && h > 0 {
		hwpH = baseW * h / w
	}
	setContainerImage(area, imgID, hwpW, hwpH)
	return area
}

// setContainerImage 는 container 단락의 img 참조와 크기를 교체한다.
func setContainerImage(p *etree.Element, imgID string, w, h int) {
	ws, hs := strconv.Itoa(w), strconv.Itoa(h)
	hwpx.Walk(p, func(e *etree.Element) {
		switch e.Tag {
		case "img":
			e.CreateAttr("binaryItemIDRef", imgID)
		case "orgSz", "sz":
			e.CreateAttr("width", ws)
			e.CreateAttr("height", hs)
		case "imgRect":
			setImgRect(e, w, h)
		case "imgClip":
			e.CreateAttr("left", "0")
			e.CreateAttr("top", "0")
			e.CreateAttr("right", ws)
			e.CreateAttr("bottom", hs)
		}
	})
}

// setImgRect 는 이미지 사각형 좌표(pt0~3)를 0~w, 0~h 로 재설정한다.
func setImgRect(e *etree.Element, w, h int) {
	pts := [4][2]int{{0, 0}, {w, 0}, {w, h}, {0, h}}
	i := 0
	for _, c := range e.ChildElements() {
		if strings.HasPrefix(c.Tag, "pt") && i < 4 {
			c.CreateAttr("x", strconv.Itoa(pts[i][0]))
			c.CreateAttr("y", strconv.Itoa(pts[i][1]))
			i++
		}
	}
}

func pathExt(p string) string {
	if i := strings.LastIndex(p, "."); i >= 0 {
		return p[i:]
	}
	return ""
}

// buildTableGrid 는 표 단락(p with tbl)을 복제해 헤더행·데이터행을 가변 재구성한다.
func buildTableGrid(ref *etree.Element, t *ir.Table, ctx *buildCtx) *etree.Element {
	r := ctx.r
	p := hwpx.ClonePara(ref)
	tbl := findFirst(p, "tbl")
	if tbl == nil {
		return p
	}
	trs := childElems(tbl, "tr")
	if len(trs) < 1 {
		return p
	}
	headerTpl := trs[0].Copy()
	dataTpl := trs[0].Copy()
	if len(trs) >= 2 {
		dataTpl = trs[1].Copy()
	}
	for _, tr := range trs {
		tbl.RemoveChild(tr)
	}

	ncols := len(t.Headers)
	if ncols == 0 && len(t.Rows) > 0 {
		ncols = len(t.Rows[0])
	}
	if ncols < 1 {
		ncols = 1
	}
	widths := colWidths(r.Table.Width, ncols)

	rowIdx := 0
	if len(t.Headers) > 0 {
		tbl.AddChild(buildRow(headerTpl, rowIdx, t.Headers, widths, r.Table.HeaderBorderFill, r.Table.HeaderHeight))
		rowIdx++
	}
	for _, row := range t.Rows {
		tbl.AddChild(buildRow(dataTpl, rowIdx, row, widths, r.Table.DataBorderFill, r.Table.BodyHeight))
		rowIdx++
	}

	tbl.CreateAttr("rowCnt", strconv.Itoa(rowIdx))
	tbl.CreateAttr("colCnt", strconv.Itoa(ncols))
	if sz := findChild(tbl, "sz"); sz != nil {
		hgt := 0
		if len(t.Headers) > 0 {
			hgt += r.Table.HeaderHeight
		}
		hgt += r.Table.BodyHeight * len(t.Rows)
		sz.CreateAttr("width", strconv.Itoa(r.Table.Width))
		sz.CreateAttr("height", strconv.Itoa(hgt))
	}
	hwpx.RemoveLineSegArray(p)
	return p
}

func buildRow(tpl *etree.Element, rowIdx int, cells []string, widths []int, bf string, height int) *etree.Element {
	tr := tpl.Copy()
	tcs := childElems(tr, "tc")
	if len(tcs) == 0 {
		return tr
	}
	tcTpl := tcs[0].Copy()
	for _, tc := range tcs {
		tr.RemoveChild(tc)
	}
	for col := 0; col < len(widths); col++ {
		tc := tcTpl.Copy()
		if bf != "" {
			tc.CreateAttr("borderFillIDRef", bf)
		}
		if ca := findChild(tc, "cellAddr"); ca != nil {
			ca.CreateAttr("colAddr", strconv.Itoa(col))
			ca.CreateAttr("rowAddr", strconv.Itoa(rowIdx))
		}
		if csz := findChild(tc, "cellSz"); csz != nil {
			csz.CreateAttr("width", strconv.Itoa(widths[col]))
			csz.CreateAttr("height", strconv.Itoa(height))
		}
		val := ""
		if col < len(cells) {
			val = cells[col]
		}
		setCellText(tc, val)
		tr.AddChild(tc)
	}
	return tr
}

func setCellText(tc *etree.Element, text string) {
	var firstP *etree.Element
	hwpx.Walk(tc, func(e *etree.Element) {
		if firstP == nil && e.Tag == "p" {
			firstP = e
		}
	})
	if firstP != nil {
		hwpx.SetParaText(firstP, text)
	}
}

// buildChapterToc 는 간지 미니목차 표(1열)에 소절 목록을 채운다.
func buildChapterToc(ref *etree.Element, sectionTitles []string) *etree.Element {
	if len(sectionTitles) == 0 {
		return nil
	}
	p := hwpx.ClonePara(ref)
	tbl := findFirst(p, "tbl")
	if tbl == nil {
		return p
	}
	trs := childElems(tbl, "tr")
	if len(trs) == 0 {
		return p
	}
	rowTpl := trs[0].Copy()
	for _, tr := range trs {
		tbl.RemoveChild(tr)
	}
	for ri, title := range sectionTitles {
		tr := rowTpl.Copy()
		for _, tc := range childElems(tr, "tc") {
			if ca := findChild(tc, "cellAddr"); ca != nil {
				ca.CreateAttr("rowAddr", strconv.Itoa(ri))
			}
			setTocCellText(tc, title)
		}
		tbl.AddChild(tr)
	}
	tbl.CreateAttr("rowCnt", strconv.Itoa(len(sectionTitles)))
	hwpx.RemoveLineSegArray(p)
	return p
}

func setTocCellText(tc *etree.Element, title string) {
	var t *etree.Element
	hwpx.Walk(tc, func(e *etree.Element) {
		if t == nil && e.Tag == "t" {
			t = e
		}
	})
	if t == nil {
		return
	}
	var tab *etree.Element
	for _, c := range t.ChildElements() {
		if c.Tag == "tab" {
			tab = c.Copy()
			break
		}
	}
	t.Child = nil
	t.CreateCharData(title)
	if tab != nil {
		t.AddChild(tab)
		t.CreateCharData("00")
	}
}

func colWidths(total, ncols int) []int {
	if ncols < 1 {
		ncols = 1
	}
	base := total / ncols
	w := make([]int, ncols)
	for i := range w {
		w[i] = base
	}
	w[ncols-1] = total - base*(ncols-1)
	return w
}

func childElems(e *etree.Element, tag string) []*etree.Element {
	var out []*etree.Element
	for _, c := range e.ChildElements() {
		if c.Tag == tag {
			out = append(out, c)
		}
	}
	return out
}

func findChild(e *etree.Element, tag string) *etree.Element {
	for _, c := range e.ChildElements() {
		if c.Tag == tag {
			return c
		}
	}
	return nil
}

func findFirst(root *etree.Element, tag string) *etree.Element {
	var f *etree.Element
	hwpx.Walk(root, func(e *etree.Element) {
		if f == nil && e.Tag == tag {
			f = e
		}
	})
	return f
}
