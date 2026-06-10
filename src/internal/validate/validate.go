// Package validate 는 완성된 HWPX 의 무결성을 검사한다.
package validate

import (
	"fmt"
	"regexp"
	"strings"

	"hwpx-uri-gen/internal/hwpx"
)

// 본문에 남으면 안 되는 placeholder 패턴(미치환 = 조립 누락).
var placeholderRe = regexp.MustCompile(
	`\[보고서 제목|\[연구실명\]|\[연구자명\]|\[내용\]|\[소절제목\]|\[소절 제목\]|` +
		`\[챕터 제목\]|\[쳅터제목\]|\[세부 항목 제목\]|\[상세 항목 제목\]|` +
		`\[주요 항목|\[세부 설명|\[항목\d\]|\[그림 영역|\[표 제목\]|\[그림 제목\]|\[부록`)

// Validate 는 hwpx 파일을 검사해 치명 오류와 경고를 반환한다.
func Validate(path string) (crit []string, warn []string) {
	pkg, err := hwpx.Open(path)
	if err != nil {
		return []string{"ZIP 열기 실패: " + err.Error()}, nil
	}

	// mimetype
	if mt, ok := pkg.Read("mimetype"); !ok || string(mt) != "application/hwp+zip" {
		warn = append(warn, "mimetype 이 application/hwp+zip 이 아님")
	}

	for _, name := range []string{"Contents/section0.xml", "Contents/section1.xml"} {
		data, ok := pkg.Read(name)
		if !ok {
			crit = append(crit, name+" 누락")
			continue
		}
		s := string(data)

		// placeholder 잔여 (치명)
		if m := placeholderRe.FindAllString(s, -1); len(m) > 0 {
			u := uniq(m)
			n := len(u)
			if n > 3 {
				n = 3
			}
			crit = append(crit, fmt.Sprintf("%s placeholder 미치환 %d건: %v…", name, len(m), u[:n]))
		}

		// 네임스페이스 깨짐 (치명) — etree 사용 시 발생 안 해야 정상
		if strings.Contains(s, "ns0:") {
			crit = append(crit, name+" ns0 네임스페이스 잔여(한글 빈 페이지 위험)")
		}

		// linesegarray 잔여 (경고) — 텍스트 바꾼 단락에 남으면 줄바꿈 깨질 수 있음
		if strings.Count(s, "linesegarray") > 0 {
			warn = append(warn, fmt.Sprintf("%s linesegarray 잔여 %d건", name, strings.Count(s, "linesegarray")))
		}
	}
	return crit, warn
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
