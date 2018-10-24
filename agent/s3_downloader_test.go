package agent

import (
	"testing"
)

func TestS3DowloaderBucketPath(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Bucket, Expected string
	}{
		{"s3://my-bucket-name/foo/bar", "foo/bar"},
		{"s3://starts-with-an-s/and-this-is-its/folder", "and-this-is-its/folder"},
	} {
		d := S3Downloader{Bucket: tc.Bucket}
		if p := d.BucketPath(); p != tc.Expected {
			t.Error("bad bucket path", p)
		}
	}
}

func TestS3DowloaderBucketName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Bucket, Expected string
	}{
		{"s3://my-bucket-name/foo/bar", "my-bucket-name"},
		{"s3://starts-with-an-s/and-this-is-its/folder", "starts-with-an-s"},
	} {
		d := S3Downloader{Bucket: tc.Bucket}
		if n := d.BucketName(); n != tc.Expected {
			t.Error("bad bucket name", n)
		}
	}
}

func TestS3DowloaderBucketFileLocation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Bucket, Path, Expected string
	}{
		{"s3://my-bucket-name/s3/folder", "here/please/right/now/", "s3/folder/here/please/right/now/"},
		{"s3://my-bucket-name/s3/folder", "", "s3/folder/"},
	} {
		d := S3Downloader{Bucket: tc.Bucket, Path: tc.Path}
		if l := d.BucketFileLocation(); l != tc.Expected {
			t.Error("bad bucket file location", l)
		}
	}
}
