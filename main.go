package main

import (
	"archive/tar"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

var xflag bool
var fflag string

func init() {
	flag.BoolVar(&xflag, "x", false, "extract from archive")
	flag.StringVar(&fflag, "f", "out.tar", "archive file name")

	flag.Parse()
}

func main() {
	if xflag {
		untar()
	} else {
		totar()
	}
}

func untar() {
	if len(flag.Args()) != 1 {
		fmt.Println("invalid args - required extracting path")
		os.Exit(1)
	}

	abstar, err := filepath.Abs(filepath.Clean(fflag))
	mustNotErr(err, fmt.Sprintf("failed to find tar %s: %s", fflag, err))

	expath, err := filepath.Abs(filepath.Clean(flag.Args()[0]))
	mustNotErr(err, fmt.Sprintf("failed to find extracting path %s: %s", fflag, err))

	tarfile, err := os.Open(abstar)
	mustNotErr(err, fmt.Sprintf("failed to open archive: %s", err))

	defer func(tarfile *os.File) {
		if err := tarfile.Close(); err != nil {
			fmt.Printf("failed to close tar file: %s\n", err)
		}
	}(tarfile)

	tr := tar.NewReader(tarfile)

	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("failed to extract file from archive: %s\n", err)
			continue
		}

		abspath := filepath.Join(expath, hdr.Name)

		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(abspath, 0775); err != nil {
				fmt.Printf("failed to extract create directiory %s: %s\n", abspath, err)
			}
			continue
		}

		file, err := os.Create(abspath)
		if err != nil {
			fmt.Printf("failed to create file %s: %s; skipping\n", abspath, err)
			continue
		}

		_, cperr := io.Copy(file, tr)

		if clserr := file.Close(); err != nil {
			fmt.Printf("failed to close file %s: %s\n", abspath, clserr)
			continue
		}

		if cperr != nil {
			fmt.Printf("failed to copy data to file %s: %s\n", abspath, err)
			continue
		}
	}
}

func totar() {
	if len(flag.Args()) == 0 {
		fmt.Println("missing files to create archive")
		os.Exit(1)
	}

	abstar, err := filepath.Abs(fflag)
	mustNotErr(err, fmt.Sprintf("failed to create tar file: %s", err))

	tarfile, err := os.Create(abstar)
	mustNotErr(err, fmt.Sprintf("failed to create tar file: %s", err))

	defer func(tarfile *os.File) {
		if err := tarfile.Close(); err != nil {
			fmt.Printf("failed to close tar file: %s\n", err)
		}
	}(tarfile)

	tw := tar.NewWriter(tarfile)
	defer func(tw *tar.Writer) {
		if err := tw.Close(); err != nil {
			fmt.Printf("failed to close tar writer: %s\n", err)
		}
	}(tw)

	for _, arg := range flag.Args() {
		basepath, err := filepath.Abs(arg)
		if err != nil {
			fmt.Printf("failed to find absolute path to argument %s: %s; skipping\n", arg, err)
			continue
		}

		walker := func(path string, d fs.DirEntry, err error) (e error) {
			if err != nil || path == abstar {
				return err
			}

			finfo, err := d.Info()
			if err != nil {
				return err
			}

			hdr, err := tar.FileInfoHeader(finfo, d.Name())
			if err != nil {
				return err
			}

			hdr.Name, e = filepath.Rel(basepath, path)
			if e != nil {
				return
			}

			if e = tw.WriteHeader(hdr); e != nil {
				return
			}

			if d.IsDir() {
				return nil
			}

			src, e := os.Open(path)
			if e != nil {
				return
			}

			defer func(src *os.File) {
				if err := src.Close(); err != nil {
					e = err
				}
			}(src)

			_, e = io.Copy(tw, src)
			return
		}

		if err := filepath.WalkDir(basepath, walker); err != nil {
			fmt.Printf("failed to archive %s: %s; skipping\n", arg, err)
			continue
		}
	}
}

func mustNotErr(err error, mess string) {
	if err != nil {
		fmt.Println(mess)
		os.Exit(1)
	}
}
