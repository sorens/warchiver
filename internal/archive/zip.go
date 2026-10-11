package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	walog "github.com/sorens/warchiver/internal/log"
)

func TarGzip(ctx context.Context, src, dstDir string, manifest *Manifest, chars []*CharacterEntry, extraExt ...string) (archivePath string, retErr error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// warchive.20260313T194323Z.ee5306d5.tar.gz
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", err
	}

	// cache a single date/time object.
	// every file will have the same timestamp which is preferred
	now := time.Now()

	hash, err := HashManifest(manifest)
	if err != nil {
		return "", err
	}

	ts := now.UTC().Format("20060102T150405Z")
	filename := fmt.Sprintf("warchive.%s.%s.tar.gz", ts, hash)
	outputPath := filepath.Join(dstDir, filename)

	target, err := os.CreateTemp(dstDir, ".warchive.*.tmp")
	if err != nil {
		return "", err
	}
	tempPath := target.Name()
	targetClosed := false
	defer func() {
		if !targetClosed {
			if err := target.Close(); err != nil && retErr == nil {
				retErr = err
			}
		}
		os.Remove(tempPath)
		if retErr != nil {
			archivePath = ""
		}
	}()

	start := time.Now()

	gzw, err := gzip.NewWriterLevel(target, gzip.BestSpeed)
	if err != nil {
		return "", err
	}
	tw := tar.NewWriter(gzw)
	tarClosed := false
	gzipClosed := false
	defer func() {
		if !tarClosed {
			tw.Close()
		}
		if !gzipClosed {
			gzw.Close()
		}
	}()

	hashes := map[string]string{}

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			return err
		}

		// Compute relative path from sourceDir
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if rel == "." || strings.HasSuffix(rel, ".DS_Store") {
			return nil
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		err = func() error {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			base := filepath.Base(path)
			isJSON := strings.HasSuffix(base, ".json")
			isJSONL := strings.HasSuffix(base, ".jsonl")
			isSIMC := strings.HasSuffix(base, ".simc")
			isHTML := strings.HasSuffix(base, ".html")
			isExtra := false
			for _, ext := range extraExt {
				if strings.HasSuffix(base, ext) {
					isExtra = true
					break
				}
			}
			if !isJSON && !isJSONL && !isSIMC && !isHTML && !isExtra {
				walog.Ctx(ctx).Verbose("skipping non-archived file: %s", path)
				return nil
			}

			var buf *bytes.Buffer
			if isJSON && base != "manifest.json" {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				reformStart := time.Now()
				buf, err = ReformatJsonToBuffer(ctx, file)
				if err != nil {
					if ctxErr := ctx.Err(); ctxErr != nil {
						return ctxErr
					}
					walog.Ctx(ctx).Debug("failed to reformat JSON, archiving original data: %s: %v", path, err)
					raw, readErr := readFileContext(ctx, path)
					if readErr != nil {
						return fmt.Errorf("failed to read original JSON after reformat failure: %w", readErr)
					}
					buf = bytes.NewBuffer(raw)
				} else {
					walog.Ctx(ctx).Verbose("%7v %7v %11v reformat %s", info.Size(), buf.Len(), time.Since(reformStart), path)
				}
			} else {
				raw, err := readFileContext(ctx, path)
				if err != nil {
					return err
				}
				buf = bytes.NewBuffer(raw)
				// walog.Ctx(ctx).Verbose("%10v %10v %10v zipping %s", info.Size(), buf.Len(), "", path)
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}

			hdr := &tar.Header{
				Name:    filepath.ToSlash(rel),
				Mode:    0644,
				Size:    int64(buf.Len()),
				ModTime: now,
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			var value string
			if !strings.HasSuffix(filepath.ToSlash(rel), "manifest.json") {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				sum, err := hashContext(ctx, crypto.SHA256.New(), bytes.NewReader(buf.Bytes()))
				if err != nil {
					return err
				}
				value = hex.EncodeToString(sum)
			}

			writeStart := time.Now()
			written, err := io.Copy(tw, &contextReader{ctx: ctx, reader: buf})
			walog.Ctx(ctx).Verbose("%7v %7v %11v gzip write %s", written, "", time.Since(writeStart), rel)
			if err == nil {
				// store the hash of this data for later use
				hashes[filepath.ToSlash(rel)] = value
			}
			return err
		}()

		return err
	})
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if err := tw.Close(); err != nil {
		tarClosed = true
		return "", fmt.Errorf("closing tar archive: %w", err)
	}
	tarClosed = true
	if err := gzw.Close(); err != nil {
		gzipClosed = true
		return "", fmt.Errorf("closing gzip archive: %w", err)
	}
	gzipClosed = true
	if err := target.Close(); err != nil {
		targetClosed = true
		return "", fmt.Errorf("closing archive file: %w", err)
	}
	targetClosed = true

	keys := make([]string, 0, len(hashes))
	for k := range hashes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		v := hashes[k]
		if v != "" {
			b.WriteString(v)
		}
		walog.Ctx(ctx).Verbose("%64v %s", v, k)
	}

	sum := sha256.Sum256([]byte(b.String()))
	walog.Ctx(ctx).Verbose("hash: %s", hex.EncodeToString(sum[:]))
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, outputPath); err != nil {
		return "", fmt.Errorf("publishing archive: %w", err)
	}
	walog.Ctx(ctx).Verbose("archive written in %v: %s", time.Since(start), filename)

	return outputPath, nil
}

func readFileContext(ctx context.Context, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, &contextReader{ctx: ctx, reader: file}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func hashContext(ctx context.Context, h hash.Hash, r io.Reader) ([]byte, error) {
	if _, err := io.Copy(h, &contextReader{ctx: ctx, reader: r}); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
