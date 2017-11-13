package testtools

import (
	"archive/tar"
	"bytes"
	"fmt"
	"github.com/pierrec/lz4"
	"github.com/x4m/wal-g/internal"
	"io"
	"os"
	"path/filepath"
)

// FileTarBall represents a tarball that is
// written to disk.
type FileTarBall struct {
	out         string
	number      int
	size        int64
	writeCloser io.WriteCloser
	tarWriter   *tar.Writer
}

// SetUp creates a new LZ4 writer, tar writer and file for
// writing bundled compressed bytes to.
func (tarBall *FileTarBall) SetUp(crypter internal.Crypter, names ...string) {
	if tarBall.tarWriter == nil {
		name := filepath.Join(tarBall.out, "part_"+fmt.Sprintf("%0.3d", tarBall.number)+".tar.lz4")
		file, err := os.Create(name)
		if err != nil {
			panic(err)
		}
		var writeCloser io.WriteCloser

		if crypter.IsUsed() {
			writeCloser, err = crypter.Encrypt(file)

			if err != nil {
				panic(err)
			}

			tarBall.writeCloser = &internal.CascadeWriteCloser{
				WriteCloser: lz4.NewWriter(file),
				Underlying:  &internal.CascadeWriteCloser{WriteCloser: writeCloser, Underlying: file},
			}
		} else {
			writeCloser = file
			tarBall.writeCloser = &internal.CascadeWriteCloser{
				WriteCloser: lz4.NewWriter(file),
				Underlying:  writeCloser,
			}
		}

		tarBall.tarWriter = tar.NewWriter(tarBall.writeCloser)
	}
}

// CloseTar closes the tar writer and file, flushing any
// unwritten data to the file before closing.
func (tarBall *FileTarBall) CloseTar() error {
	err := tarBall.tarWriter.Close()
	if err != nil {
		return err
	}

	return tarBall.writeCloser.Close()
}

// Finish alerts that compression is complete.
func (tarBall *FileTarBall) Finish(sentinelDto *internal.BackupSentinelDto) error {
	fmt.Printf("Wrote %d compressed tar files to %s.\n", tarBall.number, tarBall.out)
	return nil
}

func (tarBall *FileTarBall) Size() int64            { return tarBall.size }
func (tarBall *FileTarBall) AddSize(i int64)        { tarBall.size += i }
func (tarBall *FileTarBall) TarWriter() *tar.Writer { return tarBall.tarWriter }
func (tarBall *FileTarBall) AwaitUploads()          {}

// BufferTarBall represents a tarball that is
// written to buffer.
type BufferTarBall struct {
	number     int
	size       int64
	underlying *bytes.Buffer
	tarWriter  *tar.Writer
}

func (tarBall *BufferTarBall) SetUp(crypter internal.Crypter, args ...string) {
	tarBall.tarWriter = tar.NewWriter(tarBall.underlying)
}

func (tarBall *BufferTarBall) CloseTar() error {
	return tarBall.tarWriter.Close()
}

func (tarBall *BufferTarBall) Finish(sentinelDto *internal.BackupSentinelDto) error {
	return nil
}

func (tarBall *BufferTarBall) Size() int64 {
	return tarBall.size
}

func (tarBall *BufferTarBall) AddSize(add int64) {
	tarBall.size += add
}

func (tarBall *BufferTarBall) TarWriter() *tar.Writer {
	return tarBall.tarWriter
}

func (tarBall *BufferTarBall) AwaitUploads() {}
