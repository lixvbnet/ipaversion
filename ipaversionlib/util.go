package ipaversion

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}


const DefaultUserAgent = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.61 Mobile Safari/537.36"

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total int	// total number of bytes to download
	Count int	// current number of bytes downloaded
}

func NewWriteCounter(total int) *WriteCounter {
	return &WriteCounter{Total: total}
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Count += n
	wc.PrintProgress()
	return n, nil
}

func (wc *WriteCounter) PrintProgress() {
	// clear the line by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 50))
	s := fmt.Sprintf("Progress: %.1f%%", float64(wc.Count)/float64(wc.Total)*100)
	fmt.Printf("\r%s", s)
	if wc.Count == wc.Total {
		fmt.Println()
	}
}

func DownloadFile(url string, filepath string, userAgent string) (err error) {
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// fmt.Println("Content-Length:", resp.Header.Get("Content-Length"))
	contentLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	fmt.Println("Size:", humanize.Bytes(uint64(contentLength)))
	total, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file, with progress feedback
	counter := NewWriteCounter(total)
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	return nil
}

