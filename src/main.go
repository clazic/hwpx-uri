// hwpxgen — IR(JSON) 또는 마크다운 보고서를 울산연구원 양식 HWPX 로 변환한다.
//
// 사용:
//   hwpxgen -in report.json -o 보고서.hwpx
//   hwpxgen -in report.json                       # 출력 기본값 보고서.hwpx
//   hwpxgen -in report.json -template 양식.hwpx -rules form-rules.json
//
// 기본 템플릿·규칙은 실행파일(또는 작업디렉토리)의 references/ 에서 찾는다.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hwpx-uri-gen/internal/assemble"
	"hwpx-uri-gen/internal/ir"
	"hwpx-uri-gen/internal/mdparse"
	"hwpx-uri-gen/internal/rules"
	"hwpx-uri-gen/internal/validate"
)

func main() {
	in := flag.String("in", "", "입력 경로: .md(작성가이드 규칙) 또는 .json(IR) (필수)")
	out := flag.String("o", "보고서.hwpx", "출력 HWPX 경로")
	tmpl := flag.String("template", "", "양식 템플릿 hwpx (기본: references/<rules.template>)")
	rulesPath := flag.String("rules", "", "form-rules.json (기본: references/form-rules.json)")
	flag.Parse()

	if *in == "" {
		fmt.Fprintln(os.Stderr, "오류: -in <IR JSON> 이 필요합니다")
		flag.Usage()
		os.Exit(2)
	}

	base := resolveBase()
	if *rulesPath == "" {
		*rulesPath = filepath.Join(base, "references", "form-rules.json")
	}

	// 템플릿 기본 경로는 rules.template 에서 결정
	r, err := rules.Load(*rulesPath)
	if err != nil {
		fatal(err)
	}
	if *tmpl == "" {
		*tmpl = filepath.Join(base, "references", r.Template)
	}

	rep, err := loadReport(*in)
	if err != nil {
		fatal(err)
	}

	if err := assemble.Build(rep, *tmpl, *rulesPath, *out); err != nil {
		fatal(err)
	}
	fmt.Printf("[hwpxgen] 생성 완료: %s\n", *out)

	// 검증
	crit, warn := validate.Validate(*out)
	for _, w := range warn {
		fmt.Printf("[validate] ⚠️  %s\n", w)
	}
	if len(crit) > 0 {
		for _, c := range crit {
			fmt.Printf("[validate] ❌ %s\n", c)
		}
		os.Exit(1)
	}
	fmt.Println("[validate] ✅ 검증 통과")
}

// resolveBase 는 references/ 가 있는 기준 디렉토리를 찾는다.
// 탐색 순서: 실행파일 디렉토리 → 그 상위(bin/ 배포 대응) → 현재 작업 디렉토리.
func resolveBase() string {
	var cands []string
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		cands = append(cands, d, filepath.Dir(d))
	}
	if cwd, err := os.Getwd(); err == nil {
		cands = append(cands, cwd)
	}
	for _, c := range cands {
		if _, err := os.Stat(filepath.Join(c, "references")); err == nil {
			return c
		}
	}
	cwd, _ := os.Getwd()
	return cwd
}

// loadReport 는 확장자로 입력 형식을 판단해 IR 로 읽는다(.md → 파서, 그 외 → JSON).
func loadReport(path string) (*ir.Report, error) {
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("md 읽기 실패: %w", err)
		}
		return mdparse.Parse(data)
	}
	return ir.Load(path)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "오류:", err)
	os.Exit(1)
}
