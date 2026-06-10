// Package imagegen 은 흐름도·차트 이미지를 PNG 로 생성한다(외부 차트 라이브러리 없이).
// 폰트는 OS별 한글 폰트를 탐색해 golang.org/x/image 로 렌더링한다(크로스플랫폼).
package imagegen

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// 양식 청록(Teal) 계열 색상
var (
	colBoxFill   = color.RGBA{0xE8, 0xF2, 0xF0, 0xFF} // 연청록 박스 배경
	colBoxBorder = color.RGBA{0x2A, 0x9D, 0x8F, 0xFF} // 청록 테두리
	colText      = color.RGBA{0x1A, 0x1A, 0x1A, 0xFF} // 본문 텍스트
	colTitle     = color.RGBA{0x1F, 0x1F, 0x1F, 0xFF}
	colArrow     = color.RGBA{0x5A, 0x6B, 0x7A, 0xFF} // 화살표·연결선
	colMerge     = color.RGBA{0x2A, 0x9D, 0x8F, 0xFF}
	colWhite     = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
)

// MakeFlowchart 는 흐름도를 PNG 로 저장하고 (폭, 높이)를 반환한다.
// parallelTop: 상단에 가로로 나란히 놓고 하나로 합류시킬 박스들(선택).
// steps: 합류 이후 위→아래로 화살표로 잇는 단계 박스들.
func MakeFlowchart(outPath, title string, parallelTop, steps []string) (int, int, error) {
	const width = 1000
	boxH, gap := 70, 56
	padTop := 32
	if title != "" {
		padTop = 84
	}
	bottom := 36

	// 행 수: 병렬(있으면 1행) + 단계들
	nRows := len(steps)
	if len(parallelTop) > 0 {
		nRows++
	}
	if nRows == 0 {
		return 0, 0, fmt.Errorf("흐름도에 내용이 없음")
	}
	height := padTop + nRows*boxH + (nRows-1)*gap + bottom
	if len(parallelTop) > 0 {
		height += int(float64(gap) * 0.5) // 병렬 합류 여백
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(img, 0, 0, width, height, colWhite)

	faceTitle, err := loadFont(28)
	if err != nil {
		return 0, 0, fmt.Errorf("폰트 로드 실패: %w", err)
	}
	defer faceTitle.Close()
	faceBox, err := loadFont(21)
	if err != nil {
		return 0, 0, err
	}
	defer faceBox.Close()

	if title != "" {
		drawTextCentered(img, faceTitle, title, width/2, 44, colTitle)
	}

	cx := width / 2
	y := padTop
	var prevBottom int = -1

	// 상단 병렬 박스
	if len(parallelTop) > 0 {
		n := len(parallelTop)
		seg := width / n
		mergeY := y + boxH + int(float64(gap)*0.45)
		xs := make([]int, n)
		for i, txt := range parallelTop {
			bx := seg*i + seg/2
			xs[i] = bx
			bw := seg - 40
			drawBox(img, faceBox, txt, bx, y+boxH/2, bw, boxH)
			// 박스 하단 → 합류선
			drawVLine(img, bx, y+boxH, mergeY, colMerge, 2)
		}
		// 가로 합류선
		drawHLine(img, xs[0], xs[n-1], mergeY, colMerge, 2)
		// 합류선 중앙 → 아래로
		prevBottom = mergeY
		y = mergeY + int(float64(gap)*0.5)
	}

	// 세로 단계 박스
	for i, txt := range steps {
		boxTop := y
		boxCy := y + boxH/2
		if prevBottom >= 0 {
			drawArrow(img, cx, prevBottom, cx, boxTop, colArrow)
		}
		bw := 460
		drawBox(img, faceBox, txt, cx, boxCy, bw, boxH)
		prevBottom = y + boxH
		if i < len(steps)-1 {
			y = prevBottom + gap
		}
	}

	f, err := os.Create(outPath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return 0, 0, err
	}
	return width, height, nil
}

// drawBox 는 중앙(cx,cy) 기준 둥근 느낌의 사각 박스 + 중앙정렬 텍스트(줄바꿈)를 그린다.
func drawBox(img *image.RGBA, face font.Face, text string, cx, cy, bw, bh int) {
	x0, y0 := cx-bw/2, cy-bh/2
	x1, y1 := cx+bw/2, cy+bh/2
	fillRect(img, x0, y0, x1, y1, colBoxFill)
	drawRectBorder(img, x0, y0, x1, y1, colBoxBorder, 2)

	lines := wrapText(face, text, bw-28)
	lh := 28
	startY := cy - (len(lines)-1)*lh/2
	for i, ln := range lines {
		drawTextCentered(img, face, ln, cx, startY+i*lh, colText)
	}
}

// --- 기본 드로잉 헬퍼 ---

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawRectBorder(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA, t int) {
	for i := 0; i < t; i++ {
		drawHLine(img, x0, x1, y0+i, c, 1)
		drawHLine(img, x0, x1, y1-1-i, c, 1)
		drawVLine(img, x0+i, y0, y1, c, 1)
		drawVLine(img, x1-1-i, y0, y1, c, 1)
	}
}

func drawHLine(img *image.RGBA, x0, x1, y int, c color.RGBA, t int) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	for dy := 0; dy < t; dy++ {
		for x := x0; x <= x1; x++ {
			img.SetRGBA(x, y+dy, c)
		}
	}
}

func drawVLine(img *image.RGBA, x, y0, y1 int, c color.RGBA, t int) {
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for dx := 0; dx < t; dx++ {
		for y := y0; y <= y1; y++ {
			img.SetRGBA(x+dx, y, c)
		}
	}
}

// drawArrow 는 (x0,y0)→(x1,y1) 수직 화살표(하향)를 그린다.
func drawArrow(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	drawVLine(img, x1, y0, y1, c, 2)
	// 화살촉(아래쪽 삼각형)
	for dy := 0; dy < 9; dy++ {
		w := 6 - (6 * dy / 9)
		for dx := -w; dx <= w; dx++ {
			img.SetRGBA(x1+dx, y1-9+dy, c)
		}
	}
}

// --- 텍스트 ---

func drawTextCentered(img *image.RGBA, face font.Face, text string, cx, cy int, c color.RGBA) {
	w := font.MeasureString(face, text).Round()
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(cx-w/2, cy+face.Metrics().Ascent.Round()/2-2),
	}
	d.DrawString(text)
}

// drawTextRight 는 (x,y)를 오른쪽 끝 기준으로 텍스트를 그린다(세로 중앙).
func drawTextRight(img *image.RGBA, face font.Face, text string, x, y int, c color.RGBA) {
	w := font.MeasureString(face, text).Round()
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x-w, y+face.Metrics().Ascent.Round()/2-2),
	}
	d.DrawString(text)
}

// drawTextLeft 는 (x,y)를 왼쪽 끝 기준으로 텍스트를 그린다(세로 중앙).
func drawTextLeft(img *image.RGBA, face font.Face, text string, x, y int, c color.RGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y+face.Metrics().Ascent.Round()/2-2),
	}
	d.DrawString(text)
}

// wrapText 는 폭(maxW)에 맞춰 텍스트를 룬 단위로 줄바꿈한다(한글 대응).
func wrapText(face font.Face, text string, maxW int) []string {
	if font.MeasureString(face, text).Round() <= maxW {
		return []string{text}
	}
	var lines []string
	var cur []rune
	for _, r := range text {
		trial := string(append(cur, r))
		if font.MeasureString(face, trial).Round() > maxW && len(cur) > 0 {
			lines = append(lines, string(cur))
			cur = []rune{r}
		} else {
			cur = append(cur, r)
		}
	}
	if len(cur) > 0 {
		lines = append(lines, string(cur))
	}
	return lines
}

// --- 폰트 로딩 ---

func loadFont(size float64) (font.Face, error) {
	path := fontPath()
	if path == "" {
		return nil, fmt.Errorf("한글 폰트를 찾지 못함(OS=%s)", runtime.GOOS)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(data)
	if err != nil {
		coll, cerr := opentype.ParseCollection(data)
		if cerr != nil {
			return nil, fmt.Errorf("폰트 파싱 실패: %w", err)
		}
		f, err = coll.Font(0)
		if err != nil {
			return nil, err
		}
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

func fontPath() string {
	var cands []string
	switch runtime.GOOS {
	case "darwin":
		cands = []string{
			"/System/Library/Fonts/Supplemental/AppleGothic.ttf",
			"/System/Library/Fonts/AppleSDGothicNeo.ttc",
			"/Library/Fonts/AppleGothic.ttf",
		}
	case "windows":
		cands = []string{"C:/Windows/Fonts/malgun.ttf", "C:/Windows/Fonts/gulim.ttc"}
	default:
		cands = []string{
			"/usr/share/fonts/truetype/nanum/NanumGothic.ttf",
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJKkr-Regular.otf",
		}
	}
	for _, p := range cands {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
