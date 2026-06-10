package assemble

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/beevik/etree"
	"hwpx-uri-gen/internal/hwpx"
	"hwpx-uri-gen/internal/ir"
	"hwpx-uri-gen/internal/rules"
)

// fillFront 는 section0 의 표지 메타·발간사·요약을 IR 값으로 치환한다.
// 텍스트 패턴으로 위치를 찾으므로 양식 단락 인덱스가 바뀌어도 따라간다.
// (목차·표목차·그림목차 자동 생성은 2차 단계)
func fillFront(sec0 *etree.Element, rep *ir.Report, r *rules.Rules) {
	ps := hwpx.SecParas(sec0)

	// 시리즈 라벨: "기본정책과제" → "기본정책과제 2024-01" (series 지정 시)
	if rep.Meta.Series != "" {
		for _, p := range ps {
			if strings.TrimSpace(hwpx.ParaText(p)) == "기본정책과제" {
				hwpx.SetParaText(p, rep.Meta.Series)
				break
			}
		}
	}

	// 제목 + 연구실·연구자: style0 / para89 두 단락
	var p89 []*etree.Element
	for _, p := range ps {
		if styleParaIs(p, "0", "89") {
			p89 = append(p89, p)
		}
	}
	if len(p89) >= 1 && rep.Meta.Title != "" {
		hwpx.SetParaText(p89[0], rep.Meta.Title)
	}
	if len(p89) >= 2 {
		who := strings.TrimSpace(fmt.Sprintf("울산연구원 %s %s", rep.Meta.Lab, rep.Meta.Author))
		hwpx.SetParaText(p89[1], who)
	}

	// 원장 + 발간월: "울산연구원장" 포함 단락과 그 직전 style6 단락
	for i, p := range ps {
		if strings.HasPrefix(strings.TrimSpace(hwpx.ParaText(p)), "울산연구원장") {
			if rep.Meta.Director != "" {
				hwpx.SetParaText(p, "울산연구원장 "+rep.Meta.Director)
			}
			for j := i - 1; j >= 0; j-- {
				if ps[j].SelectAttrValue("styleIDRef", "") == "6" {
					if rep.Meta.PubDate != "" {
						hwpx.SetParaText(ps[j], rep.Meta.PubDate)
					}
					break
				}
			}
			break
		}
	}

	// 요약 4블록: 라벨 단락 다음의 style8 단락에 ○ + 내용
	fillSummaryBlock(ps, "연구목적", rep.Summary.Purpose)
	fillSummaryBlock(ps, "연구 주요", rep.Summary.Contents)
	fillSummaryBlock(ps, "결론", rep.Summary.Conclusion)
	fillSummaryBlock(ps, "정책 활용", rep.Summary.Utilization)

	// 발간사: "발간사" 라벨 다음 style8 단락에 본문을 넣고,
	// 남는 빈 단락은 제거해 발간월·원장이 같은 페이지(발간사 본문 아래)에 오도록 한다.
	// 양식은 발간사가 길 것을 전제로 빈 단락을 다수 잡아두므로, 짧은 발간사에선 과다 여백이 생긴다.
	// 발간사 미지정이면 템플릿의 기존 내용(placeholder 또는 이전 보고서 글)을 비운다.
	fwLabel := -1
	for i, p := range ps {
		if strings.TrimSpace(hwpx.ParaText(p)) == "발간사" {
			fwLabel = i
			break
		}
	}
	if fwLabel >= 0 {
		// 발간사 본문 단락(style8) 수집 — 발간월(style6) 직전까지
		var body []*etree.Element
		for j := fwLabel + 1; j < len(ps); j++ {
			s := ps[j].SelectAttrValue("styleIDRef", "")
			if s == "6" {
				break // 발간월/원장 도달
			}
			if s == "8" {
				body = append(body, ps[j])
			}
		}
		if len(body) > 0 {
			if rep.Foreword != "" {
				hwpx.SetParaText(body[0], rep.Foreword)
				const keep = 3 // 본문 1단락 + 발간월 위 여백 2단락
				for k := keep; k < len(body); k++ {
					if par := body[k].Parent(); par != nil {
						par.RemoveChild(body[k])
					}
				}
			} else {
				hwpx.SetParaText(body[0], "")
			}
		}
	}

	// 목차·표목차·그림목차 영역의 placeholder 비우기.
	// 목차 자동 생성은 미구현이므로, 양식의 "[챕터 제목]" 류가 남아 검증에 걸리지 않도록 정리한다.
	blankTocPlaceholders(ps)
}

// tocPlaceholderRe 는 목차 영역에 남는 양식 placeholder 토큰이다(validate 와 동일 계열).
var tocPlaceholderRe = regexp.MustCompile(`\[챕터 제목\]|\[쳅터제목\]|\[소절 제목\]|\[소절제목\]|\[표 제목\]|\[그림 제목\]`)

// blankTocPlaceholders 는 "목 차" 라벨 이후 단락 중 placeholder 가 든 것을 비운다.
// 완성본처럼 실제 목차 텍스트가 든 템플릿에서는 아무것도 매칭되지 않아 영향이 없다.
func blankTocPlaceholders(ps []*etree.Element) {
	start := -1
	for i, p := range ps {
		if strings.TrimSpace(hwpx.ParaText(p)) == "목 차" {
			start = i
			break
		}
	}
	if start < 0 {
		return
	}
	for _, p := range ps[start+1:] {
		if tocPlaceholderRe.MatchString(hwpx.ParaText(p)) {
			hwpx.SetParaText(p, "")
		}
	}
}

// fillSummaryBlock 은 라벨로 시작하는 단락 다음의 첫 style8 단락에 "○ 내용"을 넣는다.
// 내용 미지정이면 템플릿의 기존 내용(placeholder 또는 이전 보고서 글)을 비운다.
func fillSummaryBlock(ps []*etree.Element, label, content string) {
	for i, p := range ps {
		if strings.HasPrefix(strings.TrimSpace(hwpx.ParaText(p)), label) {
			for j := i + 1; j < len(ps); j++ {
				if ps[j].SelectAttrValue("styleIDRef", "") == "8" {
					if strings.TrimSpace(content) == "" {
						hwpx.SetParaText(ps[j], "")
					} else {
						hwpx.SetParaText(ps[j], "○ "+content)
					}
					return
				}
			}
			return
		}
	}
}

func styleParaIs(p *etree.Element, style, para string) bool {
	return p.SelectAttrValue("styleIDRef", "") == style &&
		p.SelectAttrValue("paraPrIDRef", "") == para
}
