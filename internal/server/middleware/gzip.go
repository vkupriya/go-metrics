package middleware

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

const (
	httpStatusSuccess int    = 300
	compressionAlgo   string = "gzip" // compression algorythm
)

type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	b, err := c.zw.Write(p)
	if err != nil {
		return 0, fmt.Errorf("failed to write into gzip.Writer`: %w", err)
	}
	return b, nil
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < httpStatusSuccess {
		c.w.Header().Set("Content-Encoding", compressionAlgo)
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	if err := c.zw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip.Writer: %w", err)
	}
	return nil
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip.NewReader: %w", err)
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c *compressReader) Read(p []byte) (n int, err error) {
	b, err := c.zr.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("failed to read with gzip.Reader`: %w", err)
	}
	return b, nil
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return fmt.Errorf("failed to close gzip.Reader: %w", err)
	}
	return nil
}

func Compress(h http.Handler) http.Handler {
	compr := func(w http.ResponseWriter, r *http.Request) {
		sugar := zap.L().Sugar()
		ow := w
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, compressionAlgo)
		if supportsGzip {
			cw := newCompressWriter(w)
			ow = cw
			ow.Header().Set("Content-Encoding", compressionAlgo)
			defer func() {
				if err := cw.Close(); err != nil {
					sugar.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
		}
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, compressionAlgo)
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				sugar.Debug(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer func() {
				if err := cr.Close(); err != nil {
					sugar.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
		}

		h.ServeHTTP(ow, r)
	}
	return http.HandlerFunc(compr)
}
