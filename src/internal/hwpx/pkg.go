// Package hwpx 는 HWPX(ZIP+OWPML) 패키지의 입출력과 단락 편집 헬퍼를 제공한다.
//
// HWPX 무결성 3원칙(양식_심층분석보고서 §9):
//  1. mimetype 은 STORED(무압축)·ZIP 첫 항목이어야 한다 — 깨지면 한글이 인식 못 함.
//  2. 텍스트를 수정한 단락은 linesegarray(줄바꿈 좌표 캐시)를 제거해야 macOS 뷰어가 안 깨진다.
//  3. 네임스페이스 prefix(hp/hh/hc/hs)는 보존해야 한다 — etree 는 raw 토큰을 유지하므로
//     lxml 의 ns0 재작성 문제가 없다(별도 후처리 불필요).
package hwpx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/beevik/etree"
)

// Package 는 메모리에 적재된 HWPX 패키지다. 엔트리 순서를 보존한다.
type Package struct {
	names []string          // ZIP 엔트리 순서(원본 유지)
	files map[string][]byte // 엔트리 이름 → 내용
}

// Open 은 HWPX 파일을 메모리로 읽어들인다.
func Open(path string) (*Package, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("HWPX 열기 실패: %w", err)
	}
	defer zr.Close()

	p := &Package{files: make(map[string][]byte)}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("엔트리 %s 열기 실패: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("엔트리 %s 읽기 실패: %w", f.Name, err)
		}
		p.names = append(p.names, f.Name)
		p.files[f.Name] = data
	}
	return p, nil
}

// Read 는 엔트리 내용을 반환한다.
func (p *Package) Read(name string) ([]byte, bool) {
	data, ok := p.files[name]
	return data, ok
}

// Set 은 엔트리 내용을 갱신한다(없으면 끝에 추가).
func (p *Package) Set(name string, data []byte) {
	if _, ok := p.files[name]; !ok {
		p.names = append(p.names, name)
	}
	p.files[name] = data
}

// Names 는 엔트리 이름 목록(순서 보존)을 반환한다.
func (p *Package) Names() []string { return p.names }

// Section 은 Contents/sectionN.xml 등을 etree 문서로 파싱한다.
func (p *Package) Section(name string) (*etree.Document, error) {
	data, ok := p.files[name]
	if !ok {
		return nil, fmt.Errorf("엔트리 없음: %s", name)
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		return nil, fmt.Errorf("%s XML 파싱 실패: %w", name, err)
	}
	return doc, nil
}

// SetSection 은 편집한 etree 문서를 직렬화해 엔트리에 되쓴다.
// XML 선언과 네임스페이스 prefix 는 etree 가 원본 그대로 보존한다.
func (p *Package) SetSection(name string, doc *etree.Document) error {
	data, err := doc.WriteToBytes()
	if err != nil {
		return fmt.Errorf("%s 직렬화 실패: %w", name, err)
	}
	p.Set(name, data)
	return nil
}

// Write 는 패키지를 HWPX 파일로 쓴다. mimetype 을 STORED·첫 항목으로 보장한다.
func (p *Package) Write(path string) error {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	writeEntry := func(name string, data []byte, store bool) error {
		method := zip.Deflate
		if store {
			method = zip.Store
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}

	// 1) mimetype 을 STORED 로 가장 먼저
	if data, ok := p.files["mimetype"]; ok {
		if err := writeEntry("mimetype", data, true); err != nil {
			return fmt.Errorf("mimetype 쓰기 실패: %w", err)
		}
	}
	// 2) 나머지를 원래 순서대로 Deflate
	for _, name := range p.names {
		if name == "mimetype" {
			continue
		}
		if err := writeEntry(name, p.files[name], false); err != nil {
			return fmt.Errorf("%s 쓰기 실패: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("ZIP 마감 실패: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("파일 쓰기 실패: %w", err)
	}
	return nil
}
