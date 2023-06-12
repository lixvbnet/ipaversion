package util

import (
	"bytes"
	"compress/gzip"
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"golang.org/x/exp/maps"
	"io"
	"net/http"
)

func CloneRequest(req *proxy.Request) (*http.Request, error) {
	headerClone := maps.Clone(req.Header)
	bodyClone := bytes.Clone(req.Body)
	reqClone, err := http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(bodyClone))
	reqClone.Header = headerClone
	return reqClone, err
}

func GzipDecode(p []byte) ([]byte, error) {
	return GzipDecodeReader(bytes.NewReader(p))
}

func GzipDecodeReader(r io.Reader) ([]byte, error) {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}
