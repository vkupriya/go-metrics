package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	compressionLib string = "gzip" // compression algorythm
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	size, err := w.Writer.Write(b)
	if err != nil {
		log.Printf("error in writing with gzip writer.")
		return 0, fmt.Errorf("error in writing with gzip writer: %w", err)
	}
	return size, nil
}

func GzipHandle(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, compressionLib)
		if sendsGzip {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			r.Body = gr
			defer func() {
				if err := gr.Close(); err != nil {
					log.Println(err)
					http.Error(w, "", http.StatusInternalServerError)
				}
			}()
		}

		supportsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), compressionLib)
		if supportsGzip {
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				log.Println("error creating gzip writer.")
			}
			w.Header().Set("Content-Encoding", compressionLib)
			defer func() {
				if err := gz.Close(); err != nil {
					log.Println(err)
					http.Error(w, "", http.StatusInternalServerError)
				}
			}()
			h.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}
