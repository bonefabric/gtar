package main

import (
	"archive/tar"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
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
	//todo realize
}

func totar() {
	if len(flag.Args()) == 0 {
		fmt.Println("missing files to create archive")
		os.Exit(1)
	}

	abstar, err := filepath.Abs(fflag)
	if err != nil {
		log.Printf("failed to create tar file: %s\n", err)
		os.Exit(1)
	}

	tarfile, err := os.Create(abstar)
	if err != nil {
		log.Printf("failed to create tar file: %s\n", err)
		os.Exit(1)
	}

	defer func(tarfile *os.File) {
		if err := tarfile.Close(); err != nil {
			log.Printf("failed to close tar file: %s\n", err)
		}
	}(tarfile)

	tw := tar.NewWriter(tarfile)
	defer func(tw *tar.Writer) {
		if err := tw.Close(); err != nil {
			log.Printf("failed to close tar writer: %s\n", err)
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
