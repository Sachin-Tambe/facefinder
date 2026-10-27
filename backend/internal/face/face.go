// Package face: FaceProvider abstraction (§39). Business logic never imports AWS directly.
package face

import ("context";"fmt";"hash/fnv")

type IndexedFace struct { FaceID, ImageID, BBoxJSON string; Confidence float64 }
type Detected struct { Count int }
type Match struct { FaceID string; Similarity float64 }

type Provider interface {
	CreateCollection(ctx context.Context, id string) error
	DeleteCollection(ctx context.Context, id string) error
	IndexPhoto(ctx context.Context, collection string, image []byte, externalID string) ([]IndexedFace, error)
	DetectFaces(ctx context.Context, image []byte) (Detected, error)
	SearchFace(ctx context.Context, collection string, image []byte, threshold float64, max int) ([]Match, error)
	DeleteFaces(ctx context.Context, collection string, ids []string) error
}

// FakeProvider: deterministic dev provider (no AWS). Faces = derived from image bytes,
// so the same selfie bytes that were "indexed" from a photo will match. Dev/test only.
type Fake struct{}
func digest(b []byte) uint64 { h:=fnv.New64a(); h.Write(b); return h.Sum64() }
func (Fake) CreateCollection(context.Context,string) error { return nil }
func (Fake) DeleteCollection(context.Context,string) error { return nil }
func (Fake) IndexPhoto(_ context.Context, _ string, img []byte, ext string) ([]IndexedFace,error) {
	if len(img)==0 { return nil, fmt.Errorf("empty image") }
	// Dev heuristic: every non-empty image yields one face keyed by a coarse byte-bucket.
	id := fmt.Sprintf("fake-%d", digest(img)%997)
	return []IndexedFace{{FaceID:id, ImageID:ext, BBoxJSON:`{"Left":0.1,"Top":0.1,"Width":0.5,"Height":0.5}`, Confidence:99}}, nil
}
func (Fake) DetectFaces(_ context.Context, img []byte) (Detected,error) {
	if len(img)<100 { return Detected{0}, nil }
	return Detected{1}, nil
}
func (Fake) SearchFace(_ context.Context, _ string, img []byte, _ float64, _ int) ([]Match,error) {
	return []Match{{FaceID: fmt.Sprintf("fake-%d", digest(img)%997), Similarity: 97}}, nil
}
func (Fake) DeleteFaces(context.Context,string,[]string) error { return nil }
