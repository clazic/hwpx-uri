package imagegen

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// PNGSize 는 이미지 파일의 픽셀 크기를 반환한다(실패 시 0,0).
func PNGSize(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}
