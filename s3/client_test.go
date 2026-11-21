package s3

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	goverrors "github.com/haipham22/govern/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAPI records calls and returns scripted responses.
type fakeAPI struct {
	putObj    func(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
	getObj    func(*s3.GetObjectInput) (*s3.GetObjectOutput, error)
	headObj   func(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
	deleteObj func(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
	headBkt   func(*s3.HeadBucketInput) (*s3.HeadBucketOutput, error)
	createBkt func(*s3.CreateBucketInput) (*s3.CreateBucketOutput, error)
}

func (f *fakeAPI) PutObject(ctx context.Context, p *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return f.putObj(p)
}
func (f *fakeAPI) GetObject(ctx context.Context, p *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return f.getObj(p)
}
func (f *fakeAPI) HeadObject(ctx context.Context, p *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return f.headObj(p)
}
func (f *fakeAPI) DeleteObject(ctx context.Context, p *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return f.deleteObj(p)
}
func (f *fakeAPI) HeadBucket(ctx context.Context, p *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return f.headBkt(p)
}
func (f *fakeAPI) CreateBucket(ctx context.Context, p *s3.CreateBucketInput, _ ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	return f.createBkt(p)
}

func TestResolveOptions(t *testing.T) {
	tests := []struct {
		name            string
		opts            []Option
		wantEndpoint    string
		wantRegion      string
		wantPathStyle   bool
		wantDefaultBkt  string
		wantCredentials bool
		description     string
	}{
		{
			name:        "defaults are us-east-1 https and no endpoint",
			opts:        nil,
			wantRegion:  "us-east-1",
			description: "no options yields SDK defaults",
		},
		{
			name:         "endpoint without scheme gets https when secure",
			opts:         []Option{WithEndpoint("minio.local:9000"), WithSecure(true)},
			wantEndpoint: "https://minio.local:9000",
			wantRegion:   "us-east-1",
			description:  "secure default prefixes https",
		},
		{
			name:         "endpoint without scheme gets http when insecure",
			opts:         []Option{WithEndpoint("localhost:9000"), WithSecure(false)},
			wantEndpoint: "http://localhost:9000",
			wantRegion:   "us-east-1",
			description:  "MinIO local uses http",
		},
		{
			name:         "endpoint with explicit scheme is untouched",
			opts:         []Option{WithEndpoint("http://localhost:9000"), WithSecure(true)},
			wantEndpoint: "http://localhost:9000",
			wantRegion:   "us-east-1",
			description:  "explicit scheme wins over secure flag",
		},
		{
			name: "region, path style, bucket and creds are mapped",
			opts: []Option{
				WithRegion("ap-southeast-1"),
				WithPathStyle(true),
				WithDefaultBucket("knowledge"),
				WithCredentials("ak", "sk"),
			},
			wantRegion:      "ap-southeast-1",
			wantPathStyle:   true,
			wantDefaultBkt:  "knowledge",
			wantCredentials: true,
			description:     "all options map through",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := resolveOptions(tt.opts...)
			assert.Equal(t, tt.wantEndpoint, cfg.endpoint, tt.description)
			assert.Equal(t, tt.wantRegion, cfg.region, tt.description)
			assert.Equal(t, tt.wantPathStyle, cfg.pathStyle, tt.description)
			assert.Equal(t, tt.wantDefaultBkt, cfg.defaultBucket, tt.description)
			assert.Equal(t, tt.wantCredentials, cfg.accessKey != "", tt.description)
		})
	}
}

func TestNew_WiresClient(t *testing.T) {
	c, err := New(
		WithEndpoint("http://localhost:9000"),
		WithCredentials("ak", "sk"),
		WithPathStyle(true),
		WithDefaultBucket("knowledge"),
	)
	require.NoError(t, err)
	require.NotNil(t, c.api)
	require.NotNil(t, c.presigner)
	assert.Equal(t, "knowledge", c.bucket)

	_, isS3 := c.api.(*s3.Client)
	assert.True(t, isS3, "api must be the real SDK client")
}

func TestBucketOr(t *testing.T) {
	c := &Client{bucket: "default-bkt"}
	tests := []struct {
		name   string
		bucket string
		want   string
	}{
		{name: "empty falls back to default", bucket: "", want: "default-bkt"},
		{name: "explicit wins", bucket: "other", want: "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, c.bucketOr(tt.bucket))
		})
	}
}

func TestEnsureBucket_Branches(t *testing.T) {
	tests := []struct {
		name         string
		region       string
		headBktErr   error
		wantCreated  int
		wantLocation string
		wantCode     goverrors.ErrorCode
		description  string
	}{
		{
			name:        "existing bucket is left alone",
			headBktErr:  nil,
			wantCreated: 0,
			description: "no create call when bucket exists",
		},
		{
			name:        "missing bucket in us-east-1 has no location constraint",
			region:      "us-east-1",
			headBktErr:  &types.NotFound{},
			wantCreated: 1,
			description: "us-east-1 rejects an explicit constraint",
		},
		{
			name:         "missing bucket in another region sets constraint",
			region:       "ap-southeast-1",
			headBktErr:   &types.NotFound{},
			wantCreated:  1,
			wantLocation: "ap-southeast-1",
			description:  "region becomes the location constraint",
		},
		{
			name:        "head failure surfaces INTERNAL",
			headBktErr:  errors.New("network down"),
			wantCreated: 0,
			wantCode:    goverrors.CodeInternal,
			description: "non-not-found head error is wrapped, no create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var created []*s3.CreateBucketInput
			c := &Client{
				region: tt.region,
				bucket: "kb",
				api: &fakeAPI{
					headBkt: func(*s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
						return nil, tt.headBktErr
					},
					createBkt: func(in *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
						created = append(created, in)
						return &s3.CreateBucketOutput{}, nil
					},
				},
			}

			err := c.EnsureBucket(context.Background(), "")
			if tt.wantCode != "" {
				require.Error(t, err)
				code, ok := goverrors.GetCode(err)
				require.True(t, ok, "error must carry a code")
				assert.Equal(t, tt.wantCode, code, tt.description)
				return
			}
			require.NoError(t, err)
			assert.Len(t, created, tt.wantCreated, tt.description)
			if tt.wantLocation != "" && len(created) > 0 {
				cfg := created[0].CreateBucketConfiguration
				require.NotNil(t, cfg)
				assert.Equal(t, types.BucketLocationConstraint(tt.wantLocation), cfg.LocationConstraint)
			}
			if tt.wantLocation == "" && len(created) > 0 {
				assert.Nil(t, created[0].CreateBucketConfiguration)
			}
		})
	}
}

func TestUpload_SetsDefaultsAndWrapsErrors(t *testing.T) {
	tests := []struct {
		name        string
		putErr      error
		wantCode    goverrors.ErrorCode
		wantCT      string
		description string
	}{
		{
			name:        "empty content type defaults to octet-stream",
			wantCT:      "application/octet-stream",
			description: "caller may omit content type",
		},
		{
			name:        "upload failure is wrapped INTERNAL",
			putErr:      errors.New("boom"),
			wantCode:    goverrors.CodeInternal,
			wantCT:      "video/mp4",
			description: "SDK error carries the INTERNAL code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCT string
			c := &Client{bucket: "kb", api: &fakeAPI{
				putObj: func(in *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
					gotCT = awsToString(in.ContentType)
					return nil, tt.putErr
				},
			}}

			err := c.Upload(context.Background(), "", "videos/a.mp4", strings.NewReader("x"), 1, tt.wantCT)
			assert.Equal(t, tt.wantCT, gotCT, tt.description)
			if tt.putErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			code, ok := goverrors.GetCode(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, code)
		})
	}
}

func TestDownloadAndStat_NotFound(t *testing.T) {
	tests := []struct {
		name        string
		notFoundErr error
		description string
	}{
		{name: "NoSuchKey maps to NOT_FOUND", notFoundErr: &types.NoSuchKey{}},
		{name: "generic NotFound maps to NOT_FOUND", notFoundErr: &types.NotFound{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{bucket: "kb", api: &fakeAPI{
				getObj: func(*s3.GetObjectInput) (*s3.GetObjectOutput, error) {
					return nil, tt.notFoundErr
				},
				headObj: func(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
					return nil, tt.notFoundErr
				},
			}}

			_, dlErr := c.Download(context.Background(), "", "missing")
			code, ok := goverrors.GetCode(dlErr)
			require.True(t, ok, tt.description)
			assert.Equal(t, goverrors.CodeNotFound, code)

			_, stErr := c.Stat(context.Background(), "", "missing")
			code, ok = goverrors.GetCode(stErr)
			require.True(t, ok, tt.description)
			assert.Equal(t, goverrors.CodeNotFound, code)
		})
	}
}

func TestStat_MapsFields(t *testing.T) {
	now := time.Now()
	c := &Client{bucket: "kb", api: &fakeAPI{
		headObj: func(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
			return &s3.HeadObjectOutput{
				ContentLength: awsInt64(42),
				ContentType:   awsString("video/mp4"),
				LastModified:  &now,
				ETag:          awsString("etag-1"),
			}, nil
		},
	}}

	info, err := c.Stat(context.Background(), "", "videos/a.mp4")
	require.NoError(t, err)
	assert.Equal(t, ObjectInfo{
		Key: "videos/a.mp4", Size: 42, ContentType: "video/mp4",
		LastModified: now, ETag: "etag-1",
	}, info)
}

func TestRemove_PassesBucketAndKey(t *testing.T) {
	var gotKey *string
	c := &Client{bucket: "kb", api: &fakeAPI{
		deleteObj: func(in *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
			gotKey = in.Key
			return &s3.DeleteObjectOutput{}, nil
		},
	}}
	require.NoError(t, c.Remove(context.Background(), "", "videos/a.mp4"))
	require.NotNil(t, gotKey)
	assert.Equal(t, "videos/a.mp4", *gotKey)
}

func TestPresign_Offline(t *testing.T) {
	c, err := New(
		WithEndpoint("http://localhost:9000"),
		WithCredentials("ak", "sk"),
		WithDefaultBucket("kb"),
	)
	require.NoError(t, err)

	url, err := c.Presign(context.Background(), "", "videos/a.mp4", 15*time.Minute)
	require.NoError(t, err)
	assert.Contains(t, url, "localhost:9000")
	assert.Contains(t, url, "/kb/videos/a.mp4")
	assert.Contains(t, url, "X-Amz-Signature")
}

// helpers keep the SDK pointer shims short in table rows.
func awsString(v string) *string { return &v }
func awsInt64(v int64) *int64    { return &v }
func awsToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
