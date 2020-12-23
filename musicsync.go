package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	srcDir = flag.String("src", "", "Source directory")
	dstDir = flag.String("dst", "./", "Destination directory")
	dryRun = flag.Bool("dry", true, "Dry run")
	// This is needed to run on MTP mounted devices, which don't support move.
	useTempFile = flag.Bool("tempfile", true, "Use a temp file for atomic moves")
)

func run(inctx context.Context, srcDir, dstDir string, dry bool) error {
	if fi, err := os.Stat(srcDir); err != nil {
		return err
	} else if !fi.Mode().IsDir() {
		return errors.New("Src " + srcDir + " is not a dir")
	}
	if fi, err := os.Stat(dstDir); err != nil {
		return err
	} else if !fi.Mode().IsDir() {
		return errors.New("Dst " + dstDir + " is not a dir")
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(inctx)
	lim := make(chan struct{}, 8)
	convertErrs := make(chan error, 8)
	err := filepath.Walk(srcDir, func(srcPath string, sfi os.FileInfo, prevErr error) error {
		if prevErr != nil {
			return prevErr
		}

		var err error
		if !sfi.Mode().IsRegular() {
			return nil
		}
		relSrcPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return err
		}
		dstPathOldExt := filepath.Join(dstDir, relSrcPath)
		oldExt := filepath.Ext(dstPathOldExt)
		lowerOldExt := strings.ToLower(oldExt)
		switch lowerOldExt {
		case ".mp3":
		case ".mp4":
		case ".flac":
		case ".wma":
		case ".ogg":
		case ".opus":
		case ".m4b":
		case ".webm":
		case ".wav":
		case ".mkv":

		default:
			switch lowerOldExt {
			case ".jpg":
			case ".jpeg":
			case ".png":
			case ".cue":
			case ".nfo":
			case ".pdf":
			case ".db":
			case ".bmp":
			case ".m3u":
			case ".md5":
			case ".lnk":
			case ".gif":
			case ".htm":
			case ".url":
			case ".log":
			case ".ini":
			case ".txt":
			case ".sfv":
			default:
				log.Println("Ignoring " + relSrcPath)
			}

			return nil
		}

		dstPath := strings.TrimSuffix(dstPathOldExt, oldExt) + ".mp3"
		// FAT32 limitations:
		dstPath = strings.Replace(dstPath, "?", "_ques_", -1)

		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			log.Println("Starting " + dstPath)
			if !dry {
				wg.Add(1)
				go func() {
					if err := convert(ctx, srcPath, dstPath, dstDir, lim); err != nil {
						cancel()
						select {
						case convertErrs <- err:
						default:
						}
					}
					wg.Done()
				}()
			}
		} else if err != nil {
			return err
		}

		return nil
	})
	wg.Wait()
	if err != nil {
		return err
	}
	select {
	case err = <-convertErrs:
		return err
	default:
	}
	return nil
}

func convert(ctx context.Context, srcPath, dstPath, dstRootDir string, lim chan struct{}) error {
	select {
	case lim <- struct{}{}:
	case <-ctx.Done():
		return nil
	}
	defer func() {
		<-lim
	}()
	if err := os.MkdirAll(filepath.Dir(dstPath), 0775); err != nil {
		return err
	}
	var dst string
	if *useTempFile {
		tf, err := ioutil.TempFile(dstRootDir, "converting")
		if err != nil {
			return err
		}
		if err := tf.Close(); err != nil {
			return err
		}
		defer os.Remove(tf.Name())
		dst = tf.Name()
	} else {
		dst = dstPath
	}

	args := []string{
		"-i", srcPath,
		"-codec:a", "libmp3lame",
		"-q:a", "0",
		"-f", "mp3",
		"-y",
		dst,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	buf := &bytes.Buffer{}
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil {
		log.Println("Failure \n", err, string(buf.Bytes()))
		return err
	}

	if *useTempFile {
		if err := os.Rename(dst, dstPath); err != nil {
			return err
		}
	}

	if fi, err := os.Stat(dstPath); err != nil {
		return err
	} else {
		log.Println("Finished", fi.Name())
	}

	return nil
}

func main() {
	flag.Parse()
	if err := run(context.Background(), *srcDir, *dstDir, *dryRun); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
