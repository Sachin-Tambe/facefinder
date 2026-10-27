package face

import (
	"context"; "encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	cfg "facefinder/internal/config"
)

type AWSProvider struct{ c *rekognition.Client }

func NewAWS(ctx context.Context, c cfg.Config) (*AWSProvider, error) {
	opt := awsconfig.WithRegion(c.AWSRegion)
	var lc aws.Config; var err error
	if c.AWSKey != "" { lc, err = awsconfig.LoadDefaultConfig(ctx, opt, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AWSKey, c.AWSSecret, ""))) } else { lc, err = awsconfig.LoadDefaultConfig(ctx, opt) }
	if err != nil { return nil, err }
	return &AWSProvider{rekognition.NewFromConfig(lc)}, nil
}
func (a *AWSProvider) CreateCollection(ctx context.Context, id string) error { _, e := a.c.CreateCollection(ctx, &rekognition.CreateCollectionInput{CollectionId: &id}); return e }
func (a *AWSProvider) DeleteCollection(ctx context.Context, id string) error { _, e := a.c.DeleteCollection(ctx, &rekognition.DeleteCollectionInput{CollectionId: &id}); return e }
func (a *AWSProvider) IndexPhoto(ctx context.Context, col string, img []byte, ext string) ([]IndexedFace, error) {
	out, err := a.c.IndexFaces(ctx, &rekognition.IndexFacesInput{CollectionId: &col, ExternalImageId: &ext, Image: &types.Image{Bytes: img}, DetectionAttributes: []types.Attribute{types.AttributeDefault}, MaxFaces: aws.Int32(100), QualityFilter: types.QualityFilterAuto})
	if err != nil { return nil, err }
	var faces []IndexedFace
	for _, r := range out.FaceRecords {
		bb, _ := json.Marshal(r.Face.BoundingBox)
		faces = append(faces, IndexedFace{FaceID: aws.ToString(r.Face.FaceId), ImageID: aws.ToString(r.Face.ImageId), BBoxJSON: string(bb), Confidence: float64(aws.ToFloat32(r.Face.Confidence))})
	}
	return faces, nil
}
func (a *AWSProvider) DetectFaces(ctx context.Context, img []byte) (Detected, error) {
	out, err := a.c.DetectFaces(ctx, &rekognition.DetectFacesInput{Image: &types.Image{Bytes: img}})
	if err != nil { return Detected{}, err }
	return Detected{len(out.FaceDetails)}, nil
}
func (a *AWSProvider) SearchFace(ctx context.Context, col string, img []byte, thr float64, max int) ([]Match, error) {
	out, err := a.c.SearchFacesByImage(ctx, &rekognition.SearchFacesByImageInput{CollectionId: &col, Image: &types.Image{Bytes: img}, FaceMatchThreshold: aws.Float32(float32(thr)), MaxFaces: aws.Int32(int32(max))})
	if err != nil { return nil, err }
	var m []Match
	for _, f := range out.FaceMatches { m = append(m, Match{FaceID: aws.ToString(f.Face.FaceId), Similarity: float64(aws.ToFloat32(f.Similarity))}) }
	return m, nil
}
func (a *AWSProvider) DeleteFaces(ctx context.Context, col string, ids []string) error {
	if len(ids)==0 { return nil }
	_, e := a.c.DeleteFaces(ctx, &rekognition.DeleteFacesInput{CollectionId: &col, FaceIds: ids}); return e
}
var _ Provider = (*AWSProvider)(nil); var _ Provider = Fake{}
