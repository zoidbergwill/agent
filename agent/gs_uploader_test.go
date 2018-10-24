package agent

import (
	"testing"
)

func TestGSUploaderBucketPath(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Destination, Expected string
	}{
		{"gs://my-bucket-name/foo/bar", "foo/bar"},
		{"gs://starts-with-an-s/and-this-is-its/folder", "and-this-is-its/folder"},
	} {
		gsUploader := GSUploader{Destination: tc.Destination}
		if p := gsUploader.BucketPath(); p != tc.Expected {
			t.Error("bad bucket path", p)
		}
	}
}

func TestGSUploaderBucketName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Destination, Expected string
	}{
		{"gs://my-bucket-name/foo/bar", "my-bucket-name"},
		{"gs://starts-with-an-s/and-this-is-its/folder", "starts-with-an-s"},
	} {
		gsUploader := GSUploader{Destination: tc.Destination}
		if p := gsUploader.BucketName(); p != tc.Expected {
			t.Error("bad bucket name", p)
		}
	}
}
