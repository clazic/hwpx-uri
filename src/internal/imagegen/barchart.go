package imagegen

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sort"
)

// 막대그래프 계열 색상(양식 청록 계열 + 보조색)
var barPalette = []color.RGBA{
	{0x2A, 0x9D, 0x8F, 0xFF}, // 청록
	{0xE0, 0x8A, 0x3C, 0xFF}, // 주황
	{0x5A, 0x93, 0x67, 0xFF}, // 녹
	{0xB0, 0x5A, 0x7A, 0xFF}, // 자주
	{0x8A, 0x6F, 0xB0, 0xFF}, // 보라
	{0xB8, 0xCD, 0xD9, 0xFF}, // 연청회색
}

var (
	colAxis = color.RGBA{0x99, 0x99, 0x99, 0xFF}
	colGrid = color.RGBA{0xEE, 0xEE, 0xEE, 0xFF}
	colTick = color.RGBA{0x88, 0x88, 0x88, 0xFF}
)

// MakeBarChart 는 그룹 막대그래프를 PNG 로 저장하고 (폭, 높이)를 반환한다.
// labels: x축 그룹 라벨, series: 계열명→값들(라벨 순서와 대응).
func MakeBarChart(outPath, title string, labels []string, series map[string][]int) (int, int, error) {
	const width, height = 1000, 620
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(img, 0, 0, width, height, colWhite)

	faceTitle, err := loadFont(30)
	if err != nil {
		return 0, 0, fmt.Errorf("폰트 로드 실패: %w", err)
	}
	defer faceTitle.Close()
	faceLabel, _ := loadFont(20)
	defer faceLabel.Close()
	faceSmall, _ := loadFont(16)
	defer faceSmall.Close()

	if title != "" {
		drawTextCentered(img, faceTitle, title, width/2, 38, colTitle)
	}

	// 계열 이름 정렬(안정적 순서)
	var names []string
	for k := range series {
		names = append(names, k)
	}
	sort.Strings(names)

	mLeft, mRight, mTop, mBottom := 80, 50, 90, 110
	cw := width - mLeft - mRight
	chh := height - mTop - mBottom
	x0, y0 := mLeft, height-mBottom

	// 최댓값
	vmax := 1
	for _, vs := range series {
		for _, v := range vs {
			if v > vmax {
				vmax = v
			}
		}
	}
	// 10단위 올림
	vmax = ((vmax / 10) + 1) * 10

	// 축
	drawVLine(img, x0, mTop, y0, colAxis, 2)
	drawHLine(img, x0, x0+cw, y0, colAxis, 2)

	// 가로 눈금 + 그리드
	step := vmax / 5
	if step < 1 {
		step = 1
	}
	for g := 0; g <= vmax; g += step {
		gy := y0 - chh*g/vmax
		drawHLine(img, x0, x0+cw, gy, colGrid, 1)
		drawTextRight(img, faceSmall, fmt.Sprintf("%d", g), x0-12, gy, colTick)
	}

	nGroups := len(labels)
	if nGroups < 1 {
		nGroups = 1
	}
	nSeries := len(names)
	if nSeries < 1 {
		nSeries = 1
	}
	groupW := cw / nGroups
	barW := int(float64(groupW) * 0.7 / float64(nSeries))
	if barW < 4 {
		barW = 4
	}

	for gi, lab := range labels {
		gx := x0 + groupW*gi + int(float64(groupW)*0.15)
		for si, name := range names {
			vs := series[name]
			v := 0
			if gi < len(vs) {
				v = vs[gi]
			}
			bh := chh * v / vmax
			bx := gx + barW*si
			col := barPalette[si%len(barPalette)]
			fillRect(img, bx, y0-bh, bx+barW, y0, col)
			drawTextCentered(img, faceSmall, fmt.Sprintf("%d", v), bx+barW/2, y0-bh-12, colText)
		}
		drawTextCentered(img, faceLabel, lab, x0+groupW*gi+groupW/2, y0+26, colText)
	}

	// 범례
	lx, ly := x0, height-46
	for si, name := range names {
		col := barPalette[si%len(barPalette)]
		fillRect(img, lx, ly, lx+24, ly+18, col)
		drawTextLeft(img, faceLabel, name, lx+30, ly+9, colText)
		lx += 30 + len([]rune(name))*16 + 60
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
