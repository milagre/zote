package zapi

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"slices"
	"strings"

	"github.com/milagre/zote/go/zfunc"
)

type Middleware func(req Request, next HandleFunc) ResponseBuilder

func NewCORSMiddleware() Middleware {
	return func(req Request, next HandleFunc) ResponseBuilder {
		resp := next(req)

		headers := resp.Headers()
		headers.Add("Access-Control-Allow-Origin", "*")
		headers.Add("Access-Control-Allow-Method", "*")
		headers.Add("Access-Control-Allow-Headers", "*")

		return BasicResponseReader(resp.Status(), headers, resp.Body())
	}
}

func NewCompressionMiddleware() Middleware {
	return func(req Request, next HandleFunc) ResponseBuilder {
		resp := next(req)
		headers := resp.Headers()

		if headers.Get("Content-Encoding") != "" {
			return resp
		}

		encodings := zfunc.Map(
			strings.Split(req.Header().Get("Accept-Encoding"), ","),
			strings.TrimSpace,
		)

		indexGzip := slices.Index(encodings, "gzip")
		indexDeflate := slices.Index(encodings, "deflate")

		indexes := []int{}
		for _, i := range []int{indexGzip, indexDeflate} {
			if i != -1 {
				indexes = append(indexes, i)
			}
		}

		if len(indexes) == 0 {
			return resp
		}

		var ce string
		body := resp.Body()

		first := slices.Min(indexes)
		switch first {
		case indexGzip:
			ce = "gzip"
			body = gzipReader(body)

		case indexDeflate:
			ce = "deflate"
			body = deflateReader(body)
		}

		headers.Add("Content-Encoding", ce)

		return BasicResponseReader(
			resp.Status(),
			headers,
			body,
		)
	}
}

func deflateReader(source io.Reader) io.Reader {
	return compressedReader(source, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.DefaultCompression)
	})
}

func gzipReader(source io.Reader) io.Reader {
	return compressedReader(source, func(w io.Writer) (io.WriteCloser, error) {
		return gzip.NewWriterLevel(w, gzip.DefaultCompression)
	})
}

func compressedReader(source io.Reader, compressor func(io.Writer) (io.WriteCloser, error)) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()

		compressedWriter, err := compressor(writer)
		if err != nil {
			writer.CloseWithError(err)
		}

		defer compressedWriter.Close()

		_, err = io.Copy(compressedWriter, source)
		if err != nil {
			writer.CloseWithError(err)
		}
	}()
	return reader
}
