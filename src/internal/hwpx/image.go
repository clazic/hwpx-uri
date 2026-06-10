package hwpx

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// AddImage 는 새 이미지를 BinData/ 에 추가하고 content.hpf manifest 에 등록한 뒤
// 부여된 id(예: "image5")를 반환한다. img/@binaryItemIDRef 에 이 id 를 쓰면 된다.
func (p *Package) AddImage(data []byte, ext string) (string, error) {
	doc, err := p.Section("Contents/content.hpf")
	if err != nil {
		return "", err
	}
	var manifest *etree.Element
	if root := doc.Root(); root != nil {
		Walk(root, func(e *etree.Element) {
			if manifest == nil && e.Tag == "manifest" {
				manifest = e
			}
		})
	}
	if manifest == nil {
		return "", fmt.Errorf("content.hpf 에 manifest 가 없음")
	}

	// 기존 imageN id 중 최대 번호 + 1
	max := 0
	for _, item := range manifest.ChildElements() {
		id := item.SelectAttrValue("id", "")
		if strings.HasPrefix(id, "image") {
			if n, e := strconv.Atoi(strings.TrimPrefix(id, "image")); e == nil && n > max {
				max = n
			}
		}
	}
	num := max + 1
	id := fmt.Sprintf("image%d", num)
	href := fmt.Sprintf("BinData/image%d.%s", num, ext)

	item := manifest.CreateElement("opf:item")
	item.CreateAttr("id", id)
	item.CreateAttr("href", href)
	item.CreateAttr("media-type", mediaType(ext))
	item.CreateAttr("isEmbeded", "1")

	if err := p.SetSection("Contents/content.hpf", doc); err != nil {
		return "", err
	}
	p.Set(href, data)
	return id, nil
}

func mediaType(ext string) string {
	switch strings.ToLower(ext) {
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpg"
	case "gif":
		return "image/gif"
	case "bmp":
		return "image/bmp"
	default:
		return "image/" + strings.ToLower(ext)
	}
}
