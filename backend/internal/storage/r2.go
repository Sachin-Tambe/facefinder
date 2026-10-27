package storage

import (
	"bytes"; "context"; "fmt"; "time"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cfg "facefinder/internal/config"
)

type R2 struct{ c *s3.Client; pre *s3.PresignClient; bucket string }

func NewR2(ctx context.Context, c cfg.Config) (*R2, error) {
	ep := c.R2Endpoint
	if ep == "" { ep = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.R2Account) }
	lc, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("auto"), awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.R2Key, c.R2Secret, "")))
	if err != nil { return nil, err }
	cl := s3.NewFromConfig(lc, func(o *s3.Options){ o.BaseEndpoint = &ep; o.UsePathStyle = true })
	return &R2{cl, s3.NewPresignClient(cl), c.R2Bucket}, nil
}
func (r *R2) PresignUpload(ctx context.Context, k, mime string, ttl time.Duration) (string,error) {
	o, e := r.pre.PresignPutObject(ctx, &s3.PutObjectInput{Bucket:&r.bucket, Key:&k, ContentType:&mime}, s3.WithPresignExpires(ttl)); if e!=nil {return "",e}; return o.URL, nil
}
func (r *R2) PresignDownload(ctx context.Context, k string, ttl time.Duration) (string,error) {
	o, e := r.pre.PresignGetObject(ctx, &s3.GetObjectInput{Bucket:&r.bucket, Key:&k}, s3.WithPresignExpires(ttl)); if e!=nil {return "",e}; return o.URL, nil
}
func (r *R2) Put(ctx context.Context, k string, d []byte, mime string) error { _, e := r.c.PutObject(ctx, &s3.PutObjectInput{Bucket:&r.bucket, Key:&k, Body: bytes.NewReader(d), ContentType:&mime}); return e }
func (r *R2) Get(ctx context.Context, k string) ([]byte,error) {
	o, e := r.c.GetObject(ctx, &s3.GetObjectInput{Bucket:&r.bucket, Key:&k}); if e!=nil {return nil,e}; defer o.Body.Close()
	buf := new(bytes.Buffer); _, e = buf.ReadFrom(o.Body); return buf.Bytes(), e
}
func (r *R2) Delete(ctx context.Context, k string) error { _, e := r.c.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket:&r.bucket, Key:&k}); return e }
func (r *R2) DeletePrefix(ctx context.Context, prefix string) error {
	out, e := r.c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket:&r.bucket, Prefix:&prefix}); if e!=nil {return e}
	for _, o := range out.Contents { if e:=r.Delete(ctx, aws.ToString(o.Key)); e!=nil {return e} }
	return nil
}
var _ Provider = (*R2)(nil)
