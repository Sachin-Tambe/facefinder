// Image variants. Swap imaging.Decode/Encode with govips here if libvips is required — single seam.
package service

import (
	"bytes"; "image"; _ "image/jpeg"; _ "image/png"
	"github.com/disintegration/imaging"
)
func MakeVariants(src []byte) (web, thumb, analysis []byte, w, h int, err error) {
	img, err := imaging.Decode(bytes.NewReader(src)); if err != nil { return }
	b := img.Bounds(); w, h = b.Dx(), b.Dy()
	webImg := img; if w > 1600 { webImg = imaging.Resize(img, 1600, 0, imaging.Lanczos) }
	if web, err = encode(webImg); err != nil { return }
	thumb = nil
	t := imaging.Resize(img, 400, 0, imaging.Lanczos); if thumb, err = encode(t); err != nil { return }
	analysisImg := img; if w > 2000 { analysisImg = imaging.Resize(img, 2000, 0, imaging.Lanczos) }
	analysis, err = encode(analysisImg); return
}
func encode(img image.Image) ([]byte, error) { buf := new(bytes.Buffer); err := imaging.Encode(buf, img, imaging.JPEG, imaging.JPEGQuality(82)); return buf.Bytes(), err }
