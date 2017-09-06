package walg

import (
	"archive/tar"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZeroReader generates a slice of zeroes. Used to pad
// tar in cases where length of file changes.
type ZeroReader struct{}

func (z *ZeroReader) Read(p []byte) (int, error) {
	zeroes := make([]byte, len(p))
	n := copy(p, zeroes)
	return n, nil

}


type PageHeaderData struct
{
	/* XXX LSN is member of *any* block, not only page-organized ones */
	pd_lsn uint64		/* LSN: next byte after last byte of xlog
								 * record for last change to this page */
	pd_checksum uint16	/* checksum */
	pd_flags uint16		/* flag bits, see below */
	pd_lower uint16		/* offset to start of free space */
	pd_upper uint16		/* offset to end of free space */
	pd_special uint16	/* offset to start of special space */
	pd_pagesize_version uint16
	pd_prune_xid uint32 /* oldest prunable XID, or zero if none */
}


// TarWalker walks files provided by the passed in directory
// and creates compressed tar members labeled as `part_00i.tar.lzo`.
//
// To see which files and directories are skipped, please consult
// 'structs.go'. Excluded directories will be created but their
// contents will not be included in the tar bundle.
func (bundle *Bundle) TarWalker(path string, info os.FileInfo, err error) error {
	if err != nil {
		return errors.Wrap(err, "TarWalker: walk failed")
	}

	if info.Name() == "pg_control" {
		bundle.Sen = &Sentinel{info, path}
	} else if bundle.Tb.Size() <= bundle.MinSize {
		err = HandleTar(bundle, path, info)
		if err == filepath.SkipDir {
			return err
		}
		if err != nil {
			return errors.Wrap(err, "TarWalker: handle tar failed")
		}
	} else {
		oldTB := bundle.Tb
		err := oldTB.CloseTar()
		if err != nil {
			return errors.Wrap(err, "TarWalker: failed to close tarball")
		}

		bundle.NewTarBall()
		err = HandleTar(bundle, path, info)
		if err == filepath.SkipDir {
			return err
		}
		if err != nil {
			return errors.Wrap(err, "TarWalker: handle tar failed")
		}
	}
	return nil
}

// HandleTar creates underlying tar writer and handles one given file.
// Does not follow symlinks. If file is in EXCLUDE, will not be included
// in the final tarball. EXCLUDED directories are created
// but their contents are not written to local disk.
func HandleTar(bundle TarBundle, path string, info os.FileInfo) error {
	tarBall := bundle.GetTarBall()
	fileName := info.Name()
	_, ok := EXCLUDE[info.Name()]
	tarBall.SetUp()
	tarWriter := tarBall.Tw()

	if !ok {
		hdr, err := tar.FileInfoHeader(info, fileName)
		if err != nil {
			return errors.Wrap(err, "HandleTar: could not grab header info")
		}

		hdr.Name = strings.TrimPrefix(path, tarBall.Trim())
		fmt.Println(hdr.Name)

		err = tarWriter.WriteHeader(hdr)
		if err != nil {
			return errors.Wrap(err, "HandleTar: failed to write header")
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return errors.Wrapf(err, "HandleTar: failed to open file '%s'\n", path)
			}
			lim := &io.LimitedReader{
				R: io.MultiReader(f, &ZeroReader{}),
				N: int64(hdr.Size),
			}

			_, err = io.Copy(tarWriter, lim)
			if err != nil {
				return errors.Wrap(err, "HandleTar: copy failed")
			}

			tarBall.SetSize(hdr.Size)
			f.Close()
		}
	} else if ok && info.Mode().IsDir() {
		hdr, err := tar.FileInfoHeader(info, fileName)
		if err != nil {
			return errors.Wrap(err, "HandleTar: failed to grab header info")
		}

		hdr.Name = strings.TrimPrefix(path, tarBall.Trim())
		fmt.Println(hdr.Name)

		err = tarWriter.WriteHeader(hdr)
		if err != nil {
			return errors.Wrap(err, "HandleTar: failed to write header")
		}
		return filepath.SkipDir
	}

	return nil
}
