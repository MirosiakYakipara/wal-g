package testtools

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"io"
	"io/ioutil"
	"time"
)

type mockMultiFailureError struct {
	s3manager.MultiUploadFailure
	err awserr.Error
}

func (err mockMultiFailureError) UploadID() string {
	return "mock ID"
}

func (err mockMultiFailureError) Error() string {
	return err.err.Error()
}

type MockStorage map[string]bytes.Buffer

func NewMockStorage() MockStorage {
	return make(map[string]bytes.Buffer)
}

// Mock out uploader client for S3. Includes these methods:
// Upload(*UploadInput, ...func(*s3manager.Uploader))
type mockS3Uploader struct {
	s3manageriface.UploaderAPI
	multiErr bool
	err      bool
	storage  MockStorage
	writerLimit int
}

func NewMockS3Uploader(multiErr, err bool, storage MockStorage, writerLimit int) *mockS3Uploader {
	return &mockS3Uploader{multiErr: multiErr, err: err, storage: storage, writerLimit: writerLimit}
}

func (uploader *mockS3Uploader) Upload(input *s3manager.UploadInput, f ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	if uploader.err {
		return nil, awserr.New("UploadFailed", "mock Upload error", nil)
	}

	if uploader.multiErr {
		e := mockMultiFailureError{
			err: awserr.New("UploadFailed", "multiupload failure error", nil),
		}
		return nil, e
	}

	output := &s3manager.UploadOutput{
		Location:  *input.Bucket,
		VersionID: input.Key,
	}

	var err error
	if uploader.storage == nil {
		// Discard bytes to unblock pipe.
		_, err = io.Copy(ioutil.Discard, input.Body)
	} else {
		var buf bytes.Buffer
		_, err = io.Copy(NewLimitedWriter(uploader.writerLimit, &buf), input.Body)
		uploader.storage[*input.Bucket+*input.Key] = buf
	}
	if err != nil {
		return nil, err
	}

	return output, nil
}

type LimitedWriter struct {
	limit int
	untilLimit int
	underlying io.Writer
}

func NewLimitedWriter(limit int, underlying io.Writer) *LimitedWriter {
	return &LimitedWriter{limit, limit, underlying}
}

func (writer *LimitedWriter) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		toWrite := min(writer.untilLimit, len(p))
		written, err := writer.underlying.Write(p[:toWrite])
		writer.untilLimit -= written
		n += written
		if err != nil {
			return n, err
		}
		p = p[toWrite:]
		if writer.untilLimit == 0 {
			time.Sleep(time.Duration(100000000))
			writer.untilLimit = writer.limit
		}
	}
	return
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

