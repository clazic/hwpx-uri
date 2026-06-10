// Package assemble 은 IR(Report)을 양식 템플릿에 부어 HWPX 를 생성한다.
//
// 핵심 전략: 양식의 계층별 단락을 앵커로 추출해 복제(ClonePara)하고 텍스트만 교체한다.
// 서식(폰트·색·들여쓰기)은 양식이 소유하므로 조립기가 정의하지 않는다.
// 기호·번호·쪽나눔 같은 구조 규칙만 form-rules.json 으로 주입된다.
package assemble

import (
	"fmt"

	"github.com/beevik/etree"
	"hwpx-uri-gen/internal/hwpx"
	"hwpx-uri-gen/internal/ir"
	"hwpx-uri-gen/internal/rules"
)

// Build 는 IR 을 받아 HWPX 를 outPath 에 생성한다.
func Build(rep *ir.Report, templatePath, rulesPath, outPath string) error {
	r, err := rules.Load(rulesPath)
	if err != nil {
		return err
	}
	pkg, err := hwpx.Open(templatePath)
	if err != nil {
		return err
	}

	// section1: 본문·참고문헌·부록 조립
	doc1, err := pkg.Section("Contents/section1.xml")
	if err != nil {
		return err
	}
	sec1 := hwpx.FindSec(doc1)
	if sec1 == nil {
		return fmt.Errorf("section1 에 sec 요소가 없음")
	}
	a, err := collectAnchors(sec1, r)
	if err != nil {
		return err
	}
	ctx := &buildCtx{r: r, a: a, pkg: pkg, footnotes: rep.Footnotes}
	newParas := buildBodyParas(rep, ctx)
	replaceBody(sec1, newParas)
	if err := pkg.SetSection("Contents/section1.xml", doc1); err != nil {
		return err
	}

	// section0: 표지 메타·요약·발간사
	if doc0, err := pkg.Section("Contents/section0.xml"); err == nil {
		if sec0 := hwpx.FindSec(doc0); sec0 != nil {
			fillFront(sec0, rep, r)
			// 텍스트를 치환한 단락에 줄배치 캐시가 남으면 macOS 한글에서 줄바꿈이 깨진다.
			// 한글이 다시 계산하므로 section0 전체에서 제거해도 안전하다.
			hwpx.RemoveLineSegArray(sec0)
			if err := pkg.SetSection("Contents/section0.xml", doc0); err != nil {
				return err
			}
		}
	}

	// 미리보기 텍스트 갱신(옛 양식 텍스트가 Finder/한글 미리보기에 남지 않도록)
	updatePreview(pkg, rep)

	return pkg.Write(outPath)
}

// replaceBody 는 sec 의 기존 자식을 모두 제거하고 새 단락들로 교체한다.
// secPr 은 새 간지 단락(chapterCover 복제본)에 포함돼 있으므로 보존된다.
func replaceBody(sec *etree.Element, newParas []*etree.Element) {
	for _, c := range sec.ChildElements() {
		sec.RemoveChild(c)
	}
	for _, p := range newParas {
		sec.AddChild(p)
	}
}

// updatePreview 는 Preview/PrvText.txt 를 보고서 제목·요약으로 갱신한다.
func updatePreview(pkg *hwpx.Package, rep *ir.Report) {
	if _, ok := pkg.Read("Preview/PrvText.txt"); !ok {
		return
	}
	lines := "기본정책과제\n\n" + rep.Meta.Title + "\n"
	if rep.Meta.Lab != "" || rep.Meta.Author != "" {
		lines += "울산연구원 " + rep.Meta.Lab + " " + rep.Meta.Author + "\n"
	}
	if rep.Summary.Purpose != "" {
		lines += "\n요약\n" + rep.Summary.Purpose + "\n"
	}
	pkg.Set("Preview/PrvText.txt", []byte(lines))
}
