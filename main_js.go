// +build js

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gopherjs/gopherjs/js"
)

var base = js.Global.Get("location").Get("hash").String()[1:]

func getPlaylist(base string) ([]*PlaylistEntry, error) {
	resp, err := http.Get(base)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list []*PlaylistEntry

	err = json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func getReaderAt(base string, e *PlaylistEntry) (ReaderAtCloser, error) {
	return httpReaderAt(base + e.Name), nil
}

type httpReaderAt string

func (path httpReaderAt) ReadAt(b []byte, off int64) (n int, err error) {
	req, err := http.NewRequest("GET", string(path), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, off+int64(len(b))-1))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return 0, io.EOF
	}
	if resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("http: %s", resp.Status)
	}

	return io.ReadFull(resp.Body, b)
}

func (path httpReaderAt) Close() error {
	return nil
}
