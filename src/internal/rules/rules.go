// Package rules 는 form-rules.json(양식 변환 규칙)을 읽어 조립기에 제공한다.
//
// 규칙(계층별 스타일 ID·기호·번호 형식·쪽나눔)을 코드가 아니라 데이터로 둔다.
// 양식이 개정되면 form-rules.json 과 템플릿 hwpx 만 교체하면 되고 조립기는 그대로다.
package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// numberPrefixes 는 입력 텍스트 선두의 번호·기호 패턴이다(TrimNumber 용).
var numberPrefixes = []*regexp.Regexp{
	regexp.MustCompile(`^[ⅠⅡⅢⅣⅤⅥⅦⅧⅨⅩⅪⅫ]+\.\s*`), // Ⅰ.
	regexp.MustCompile(`^\(\d+\)\s*`),                   // (1)
	regexp.MustCompile(`^\d+\)\s*`),                     // 1)
	regexp.MustCompile(`^\d+\.\s*`),                     // 1.
	regexp.MustCompile(`^[○●◦–—\-※•]\s*`),               // ○ – • ※ -
}

// StyleRef 는 한 단락 유형의 스타일·문단속성 ID 참조다.
type StyleRef struct {
	Style string `json:"style"`
	Para  string `json:"para"`
}

// SymRef 는 기호가 붙는 단락 유형(○ – • ※)이다.
type SymRef struct {
	Style  string `json:"style"`
	Para   string `json:"para"`
	Symbol string `json:"symbol"`
}

// NumFormat 은 번호 형식이다(예: "1.", "1)", "(1)", "Ⅰ.").
type NumFormat struct {
	Roman  bool   `json:"roman"`
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

type Rules struct {
	Template string `json:"template"`

	Levels struct {
		Chapter struct {
			Cover  StyleRef `json:"cover"`
			Toc    StyleRef `json:"toc"`
			Title  struct {
				Style     string `json:"style"`
				Para      string `json:"para"`
				PageBreak bool   `json:"pageBreak"`
			} `json:"title"`
			Number string `json:"number"`
		} `json:"chapter"`
		Section struct {
			Style     string `json:"style"`
			ParaFirst string `json:"paraFirst"`
			ParaRest  string `json:"paraRest"`
			PageBreak bool   `json:"pageBreak"`
			Number    string `json:"number"`
		} `json:"section"`
		Subsec struct {
			Style  string `json:"style"`
			Para   string `json:"para"`
			Number string `json:"number"`
		} `json:"subsec"`
		Detail struct {
			Style  string `json:"style"`
			Para   string `json:"para"`
			Number string `json:"number"`
		} `json:"detail"`
		Point SymRef `json:"point"`
		Sub   SymRef `json:"sub"`
		Note  SymRef `json:"note"`
	} `json:"levels"`

	Table struct {
		Caption         StyleRef `json:"caption"`
		Grid            StyleRef `json:"grid"`
		HeaderBorderFill string  `json:"headerBorderFill"`
		DataBorderFill   string  `json:"dataBorderFill"`
		Width            int     `json:"width"`
		HeaderHeight     int     `json:"headerHeight"`
		BodyHeight       int     `json:"bodyHeight"`
		CaptionFmt       string  `json:"captionFmt"`
	} `json:"table"`

	Figure struct {
		Area       StyleRef `json:"area"`
		Caption    StyleRef `json:"caption"`
		CaptionFmt string   `json:"captionFmt"`
	} `json:"figure"`

	Source struct {
		Style  string `json:"style"`
		Para   string `json:"para"`
		Prefix string `json:"prefix"`
	} `json:"source"`

	Footnote struct {
		ContentCharPr string `json:"contentCharPr"`
		ContentPara   string `json:"contentPara"`
		ContentStyle  string `json:"contentStyle"`
		MarkerSuffix  string `json:"markerSuffix"`
		NumSuffix     string `json:"numSuffix"`
	} `json:"footnote"`

	References struct {
		Title StyleRef `json:"title"`
		Item  SymRef   `json:"item"`
	} `json:"references"`

	Appendix struct {
		Title StyleRef `json:"title"`
	} `json:"appendix"`

	Layout struct {
		ChapterPageBreak         bool   `json:"chapterPageBreak"`
		SectionPageBreak         bool   `json:"sectionPageBreak"`
		BlankBetweenSubsec       bool   `json:"blankBetweenSubsec"`
		BlankBetweenDetail       bool   `json:"blankBetweenDetail"`
		BlankParaStyle           string `json:"blankParaStyle"`
		BlankParaPara            string `json:"blankParaPara"`
		PageNumRestartPerChapter bool   `json:"pageNumRestartPerChapter"`
	} `json:"layout"`

	Numbering map[string]NumFormat `json:"numbering"`
}

// Load 는 form-rules.json 을 읽는다.
func Load(path string) (*Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("form-rules 읽기 실패: %w", err)
	}
	var r Rules
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("form-rules 파싱 실패: %w", err)
	}
	return &r, nil
}

var roman = []string{"", "Ⅰ", "Ⅱ", "Ⅲ", "Ⅳ", "Ⅴ", "Ⅵ", "Ⅶ", "Ⅷ", "Ⅸ", "Ⅹ", "Ⅺ", "Ⅻ"}

// Format 은 번호 형식 이름(roman_dot 등)과 카운터로 번호 문자열을 만든다.
// 예: roman_dot, 2 → "Ⅱ. " / arabic_paren, 3 → "3) " / paren_arabic, 1 → "(1) "
func (r *Rules) Format(numberKey string, n int) string {
	f, ok := r.Numbering[numberKey]
	if !ok {
		return ""
	}
	var num string
	if f.Roman {
		if n > 0 && n < len(roman) {
			num = roman[n]
		} else {
			num = fmt.Sprintf("%d", n)
		}
	} else {
		num = fmt.Sprintf("%d", n)
	}
	return f.Prefix + num + f.Suffix
}

// RomanNum 은 장 간지용 로마 숫자(접미사 없는 "Ⅰ")를 반환한다.
func (r *Rules) RomanNum(n int) string {
	if n > 0 && n < len(roman) {
		return roman[n]
	}
	return fmt.Sprintf("%d", n)
}

// TrimNumber 는 입력 텍스트 선두에 사용자가 붙인 번호·기호를 제거한다.
// IR 의 text 에 "1. 제목"처럼 번호가 들어와도 조립기가 자동 부여하므로 중복을 막는다.
func TrimNumber(text string) string {
	s := strings.TrimSpace(text)
	// 로마자/아라비아/괄호 번호 및 불릿 기호 선두 제거
	for _, pat := range numberPrefixes {
		if loc := pat.FindStringIndex(s); loc != nil && loc[0] == 0 {
			return strings.TrimSpace(s[loc[1]:])
		}
	}
	return s
}
