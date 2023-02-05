package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var xflag bool
var fflag string

func init() {
	flag.BoolVar(&xflag, "x", false, "extract from archive")
	flag.StringVar(&fflag, "f", "out.tar.gz", "archive file name")

	flag.Parse()
}

func main() {
	var op func() error
	if xflag {
		op = untar
	} else {
		op = totar
	}

	if err := op(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func totar() (err error) {
	if len(flag.Args()) == 0 {
		fmt.Println("missing files to create archive")
		os.Exit(1)
	}

	abstar, err := filepath.Abs(fflag)
	if err != nil {
		return fmt.Errorf("failed to find absolute path to archive file: %s", err)
	}

	tarfile, err := os.Create(abstar)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %s", err)
	}

	defer func(tarfile *os.File) {
		if e := tarfile.Close(); e != nil && err == nil {
			err = fmt.Errorf("failed to close archive file: %s", e)
		}
	}(tarfile)

	tw := tar.NewWriter(tarfile)
	defer func(tw *tar.Writer) {
		if e := tw.Close(); e != nil && err == nil {
			err = fmt.Errorf("failed to close archive writer: %s", e)
		}
	}(tw)

	if strings.HasSuffix(filepath.Base(abstar), ".gzip") || strings.HasSuffix(filepath.Base(abstar), ".gz") {
		zw := gzip.NewWriter(tarfile)
		defer func(zw *gzip.Writer) {
			if e := zw.Close(); e != nil && err == nil {
				err = fmt.Errorf("failed to close gzip writer: %s", e)
			}
		}(zw)

		tw = tar.NewWriter(zw)
	}

	for _, arg := range flag.Args() {
		basepath := filepath.Clean(arg)

		walker := func(path string, d fs.DirEntry, er error) (e error) {
			if er != nil {
				return er
			}

			abspath, e := filepath.Abs(path)
			if e != nil || abspath == abstar {
				return
			}

			finfo, e := d.Info()
			if e != nil {
				return
			}

			hdr, e := tar.FileInfoHeader(finfo, d.Name())
			if e != nil {
				return
			}

			relpath := path
			if filepath.IsAbs(path) {
				relpath, e = filepath.Rel(basepath, path)
				if e != nil {
					return
				}
			}
			hdr.Name = relpath

			if e = tw.WriteHeader(hdr); e != nil {
				return
			}

			if d.IsDir() {
				return
			}

			src, e := os.Open(path)
			if e != nil {
				return
			}

			defer func(src *os.File) {
				if err := src.Close(); err != nil && e == nil {
					e = err
				}
			}(src)

			_, e = io.Copy(tw, src)
			return
		}

		if err = filepath.WalkDir(basepath, walker); err != nil {
			return err
		}
	}
	return
}

func untar() (err error) {
	if len(flag.Args()) != 1 {
		return fmt.Errorf("invalid args - required extracting path")
	}

	abstar, err := filepath.Abs(filepath.Clean(fflag))
	if err != nil {
		return fmt.Errorf("failed to find archive absolute path: %s", err)
	}

	expath, err := filepath.Abs(filepath.Clean(flag.Args()[0]))
	if err != nil {
		return fmt.Errorf("failed to find extract absolute path: %s", err)
	}

	tarfile, err := os.Open(abstar)
	if err != nil {
		return fmt.Errorf("failed to open archive file: %s", err)
	}

	defer func(tarfile *os.File) {
		if err := tarfile.Close(); err != nil {
			fmt.Printf("failed to close tar file: %s\n", err)
		}
	}(tarfile)

	tr := tar.NewReader(tarfile)

	if strings.HasSuffix(filepath.Base(abstar), ".gzip") || strings.HasSuffix(filepath.Base(abstar), ".gz") {
		zr, zerr := gzip.NewReader(tarfile)
		if zerr != nil {
			return fmt.Errorf("failed to open gzip reader: %s", zerr)
		}

		defer func(zr *gzip.Reader) {
			if e := zr.Close(); e != nil && err == nil {
				err = fmt.Errorf("failed to close gzip reader: %s", e)
			}
		}(zr)
		tr = tar.NewReader(zr)
	}

	for {
		hdr, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			err = fmt.Errorf("failed to extract archive: %s", e)
			return
		}

		abspath := filepath.Join(expath, hdr.Name)

		if hdr.FileInfo().IsDir() {
			if err = os.MkdirAll(abspath, 0775); err != nil {
				return fmt.Errorf("failed to make directories: %s", err)
			}
			continue
		}

		file, err := os.OpenFile(abspath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode().Perm())
		if err != nil {
			return fmt.Errorf("failed to create file %s: %s", abspath, err)
		}

		_, cperr := io.Copy(file, tr)

		if clserr := file.Close(); clserr != nil {
			return fmt.Errorf("failed to close file %s: %s", abspath, clserr)
		}

		if cperr != nil {
			return fmt.Errorf("failed to copy data to %s: %s", abspath, cperr)
		}
	}
	return
}
