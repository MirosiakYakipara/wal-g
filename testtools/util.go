package testtools

import (
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/stretchr/testify/assert"
	"github.com/x4m/wal-g"
	"io"
	"testing"
)

func MakeDefaultUploader(
	uploaderAPI s3manageriface.UploaderAPI,
	compressor walg.Compressor,
	uploadingLocation *walg.S3Folder,
	deltaDataFolder walg.DataFolder,
	useWalDelta bool,
) *walg.Uploader {
	return walg.NewUploader(uploaderAPI, "", "", "STANDARD", compressor,
		uploadingLocation, deltaDataFolder, useWalDelta, false)
}

func NewMockUploader(apiMultiErr, apiErr bool) *walg.Uploader {
	return MakeDefaultUploader(
		NewMockS3Uploader(apiMultiErr, apiErr, nil),
		&MockCompressor{},
		walg.NewS3Folder(NewMockS3Client(true, true), "bucket/", "server", false),
		nil,
		false,
	)
}

func NewStoringMockUploader(storage *MockStorage, deltaDataFolder walg.DataFolder) *walg.Uploader {
	return MakeDefaultUploader(
		NewMockS3Uploader(false, false, storage),
		&MockCompressor{},
		walg.NewS3Folder(nil, "bucket/", "server", false),
		deltaDataFolder,
		true,
	)
}

func NewStoringCompressingMockUploader(storage *MockStorage, deltaDataFolder walg.DataFolder) *walg.Uploader {
	return MakeDefaultUploader(
		NewMockS3Uploader(false, false, storage),
		&walg.Lz4Compressor{},
		walg.NewS3Folder(NewMockStoringS3Client(storage), "bucket/", "server", true),
		deltaDataFolder,
		true,
	)
}

func NewLz4CompressingPipeWriter(input io.Reader) *walg.CompressingPipeWriter {
	return &walg.CompressingPipeWriter{
		Input: input,
		NewCompressingWriter: func(writer io.Writer) walg.ReaderFromWriteCloser {
			return walg.NewLz4ReaderFromWriter(writer)
		},
	}
}

func NewMockS3Folder(s3ClientErr, s3ClientNotFound bool) *walg.S3Folder {
	return walg.NewS3Folder(NewMockS3Client(s3ClientErr, s3ClientNotFound), "mock bucket", "mock server", false)
}

func NewStoringMockS3Folder(storage *MockStorage) *walg.S3Folder {
	return walg.NewS3Folder(NewMockStoringS3Client(storage), "bucket/", "server", false)
}

type ReadWriteNopCloser struct {
	io.ReadWriter
}

func (readWriteNopCloser *ReadWriteNopCloser) Close() error {
	return nil
}

func Contains(s *[]string, e string) bool {
	//AB: Go is sick
	if s == nil {
		return false
	}
	for _, a := range *s {
		if a == e {
			return true
		}
	}
	return false
}

func AssertReaderIsEmpty(t *testing.T, reader io.Reader) {
	buf := make([]byte, 1)
	_, err := reader.Read(buf)
	assert.Equal(t, io.EOF, err)
}

type NopCloser struct{}

func (closer *NopCloser) Close() error {
	return nil
}

type NopSeeker struct{}

func (seeker *NopSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}
