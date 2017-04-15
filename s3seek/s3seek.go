// Package s3seek provides a seekable io.Reader to an S3 object.
package s3seek

import (
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

// ReadSeekCloser implements io.Reader, io.Seeker and io.Closer.
type ReadSeekCloser interface {
	io.ReadCloser
	io.Seeker
}

type skr struct {
	oi *s3.GetObjectInput
	oo *s3.GetObjectOutput
	s3 *s3.S3
}

// Close the underlying S3 reader.
func (er *skr) Close() error {
	if er.oo == nil {
		return nil
	}
	err := er.oo.Body.Close()
	er.oo = nil
	return err

}

// Seek to the specified offset; whence must be 0.
func (er *skr) Seek(offset int64, whence int) (int64, error) {
	if whence != 0 {
		return 0, fmt.Errorf("s3seek: Seek only accepts 0 for whence argument")
	}
	var err error
	if err = er.Close(); err != nil {
		return 0, err
	}

	if err := er.setRange(offset); err != nil {
		return 0, err
	}
	return offset, nil
}

func (er *skr) setRange(offset int64) error {
	er.oi.Range = aws.String(fmt.Sprintf("bytes=%d-", offset))
	var err error
	er.oo, err = er.s3.GetObject(er.oi)
	return errors.Wrap(err, "error getting object from s3")
}

// Read implements io.Reader
func (er *skr) Read(p []byte) (int, error) {
	if er.oo == nil {
		var err error
		er.oo, err = er.s3.GetObject(er.oi)
		if err != nil {
			return 0, errors.Wrap(err, "error getting object from S3")
		}
	}
	return er.oo.Body.Read(p)
}

// New returns a io.ReadSeeker. If goi is nil then just the path is used and
// must contain the bucket prefix and the object (key) path; any s3:// prefix
// is optional. If goi is non nil, path will be ignored and the goi will be
// used to determine the bucket and key and to set any additional options.
func New(c *s3.S3, path string, goi *s3.GetObjectInput) (ReadSeekCloser, error) {
	if goi == nil {
		goi = &s3.GetObjectInput{}
		if strings.HasPrefix(path, "s3://") {
			path = path[5:]
		}

		bucketRest := strings.SplitN(path, "/", 2)
		if len(bucketRest) < 2 {
			return nil, fmt.Errorf("s3seek: expected a bucket and a key. got %s", path)
		}
		goi.Bucket = aws.String(bucketRest[0])
		goi.Key = aws.String(bucketRest[1])
	}
	er := &skr{oi: goi, s3: c}
	return er, nil
}
