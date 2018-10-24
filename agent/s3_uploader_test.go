package agent

import (
	"testing"
)

func TestS3UploaderBucketPath(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Destination, Expected string
	}{
		{"s3://my-bucket-name/foo/bar", "foo/bar"},
		{"s3://starts-with-an-s/and-this-is-its/folder", "and-this-is-its/folder"},
	} {
		u := S3Uploader{Destination: tc.Destination}
		if p := u.BucketPath(); p != tc.Expected {
			t.Error("bad bucket path", p)
		}
	}
}

func TestS3UploaderBucketName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Destination, Expected string
	}{
		{"s3://my-bucket-name/foo/bar", "my-bucket-name"},
		{"s3://starts-with-an-s/and-this-is-its/folder", "starts-with-an-s"},
	} {
		u := S3Uploader{Destination: tc.Destination}
		if n := u.BucketName(); n != tc.Expected {
			t.Error("bad bucket name", n)
		}
	}
}
