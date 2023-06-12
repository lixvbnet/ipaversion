package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/lixvbnet/sysproxy"
	"golang.org/x/exp/maps"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)
import (
	"github.com/lqqyt2423/go-mitmproxy/proxy"
)

const targetUrlSubstr = "-buy.itunes.apple.com/WebObjects/MZBuy.woa/wa/buyProduct"

var counter int64 = 0
var mu sync.Mutex		// lock when target url is matched; unlock when program exits

type ChangeHtml struct {
	proxy.BaseAddon
}

func (c *ChangeHtml) Request(f *proxy.Flow) {
	req := f.Request
	url := req.URL
	//fmt.Println("----- (On Request) ----")
	//fmt.Println(url.String())

	if strings.Contains(url.String(), targetUrlSubstr) {
		// we only want to run the function code ONCE!
		if !mu.TryLock() {
			return
		}

		headerCopy := maps.Clone(f.Request.Header)
		bodyCopy := bytes.Clone(f.Request.Body)

		clientUserAgent = headerCopy.Get("User-Agent")
		fmt.Println("User-Agent:", clientUserAgent)

		//fmt.Println("Modifying request body..")
		//f.Request.Body = bytes.ReplaceAll(f.Request.Body, []byte("<string>852645599</string>"), []byte("<string>846783668</string>"))

		// clone the request & query all versionIDs
		client := &http.Client{}
		clonedRequest, err := CloneRequest(f.Request)
		if err != nil {
			fmt.Println("Failed to clone the request!")
			log.Fatal("ERROR:", err)
		}
		res, _ := client.Do(clonedRequest)
		resBody, err := GzipDecodeReader(res.Body)
		res.Body.Close()
		if err != nil {
			fmt.Println("Failed to decode response body!")
			log.Fatal("ERROR:", err)
		}
		// get App info
		latestAppInfo := GetAppInfo(resBody)
		fmt.Println()
		fmt.Printf("[%s] (%s)\n", latestAppInfo.ItemName, latestAppInfo.ArtistName)
		fmt.Println("Latest version:\t\t", latestAppInfo.SoftwareVersionExternalIdentifier, latestAppInfo.BundleShortVersionString)
		// get all history versionIDs
		versionIDs := GetAllAppVersionIDs(resBody)
		fmt.Println("History versionIDs:\t", versionIDs)

		// replay the request
		fmt.Println("Replaying the request...")
		fmt.Println()
		// clone the request for every versionID
		//targets := []string{"842552350", "842023522", "842626028", "842726114", "844540208"}
		targets := versionIDs
		start, end := max(0, *START), min(len(targets), *END)
		for i := start; i < end; i++ {
			target := targets[i]
			atomic.AddInt64(&counter, 1)
			headerClone := maps.Clone(headerCopy)
			bodyClone := bytes.Clone(bodyCopy)
			// modify the cloned request
			strOld := fmt.Sprintf("<string>%s</string>", latestAppInfo.SoftwareVersionExternalIdentifier)
			strNew := fmt.Sprintf("<string>%s</string>", target)
			bodyClone = bytes.ReplaceAll(bodyClone, []byte(strOld), []byte(strNew))
			reqClone, _ := http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(bodyClone))
			reqClone.Header = headerClone

			resp, _ := client.Do(reqClone)

			// handle the response
			resBodyDecoded, err := GzipDecodeReader(resp.Body)
			resp.Body.Close()
			if err != nil {
				fmt.Println("Failed to decode the response body!")
				fmt.Println("ERROR:", err)
			}
			// get versionID and versionStr
			appInfo := GetAppInfo(resBodyDecoded)
			historyVersions = append(historyVersions, appInfo)
			fmt.Printf("[%d] %s %s\n", i, appInfo.SoftwareVersionExternalIdentifier, appInfo.BundleShortVersionString)
			// write the response body to file
			//err = os.WriteFile("ReplayResponse"+strconv.Itoa(int(counter))+".xml", resBodyDecoded, 0744)
			//if err != nil {
			//	fmt.Println("[ERROR]", err)
			//}
		}
		client.CloseIdleConnections()
		fmt.Println("---------------------------------------")
		fmt.Println("Done!")
		done <- true
	}
}

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

type AppInfo struct {
	BundleDisplayName string					// ex. Opera
	BundleShortVersionString string				// ex. 3.0.4
	SoftwareVersionExternalIdentifier string	// ex. 842552350
	ItemName string			// ex. Opera: 快速 &amp; 安全
	ArtistName string		// ex. Opera Software AS
	URL string				// ex. https://iosapps.itunes.apple.com/itunes-assets/../xx/yy/zz.signed.dpkg.ipa?accessKey=xxx
	ArtworkURL string		// ex. https://is4-ssl.mzstatic.com/image/thumb/Purple122/.../AppIcon-xxx.png/600x600bb.jpg
}

// All history versions
var historyVersions []*AppInfo
var clientUserAgent = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.61 Mobile Safari/537.36"

var BundleDisplayNamePattern = regexp.MustCompile(`<key>bundleDisplayName</key><string>(.*)</string>`)
var BundleShortVersionStringPattern = regexp.MustCompile(`<key>bundleShortVersionString</key><string>(.*)</string>`)
var SoftwareVersionExternalIdentifierPattern = regexp.MustCompile(`<key>softwareVersionExternalIdentifier</key><integer>(\w+)</integer>`)
var ItemNamePattern = regexp.MustCompile(`<key>itemName</key><string>(.*)</string>`)
var ArtistNamePattern = regexp.MustCompile(`<key>artistName</key><string>(.*)</string>`)
var URLPattern = regexp.MustCompile(`<key>URL</key><string>(.*)</string>`)
var ArtworkURLPattern = regexp.MustCompile(`<key>artworkURL</key><string>(.*)</string>`)

var allVersionsPattern = regexp.MustCompile(`<key>softwareVersionExternalIdentifiers</key>\s*<array>\s*([<>/\w\r\n\s]*)</array>`)
var allVersionsLinePattern = regexp.MustCompile(`<integer>(.*)</integer>`)


func GetAppInfo(data []byte) *AppInfo {
	appInfo := &AppInfo{
		BundleDisplayName:                 string(BundleDisplayNamePattern.FindSubmatch(data)[1]),
		BundleShortVersionString:          string(BundleShortVersionStringPattern.FindSubmatch(data)[1]),
		SoftwareVersionExternalIdentifier: string(SoftwareVersionExternalIdentifierPattern.FindSubmatch(data)[1]),
		ItemName:                          string(ItemNamePattern.FindSubmatch(data)[1]),
		ArtistName:                        string(ArtistNamePattern.FindSubmatch(data)[1]),
		URL:                               string(URLPattern.FindSubmatch(data)[1]),
		ArtworkURL:                        string(ArtworkURLPattern.FindSubmatch(data)[1]),
	}
	return appInfo
}

func GetAllAppVersionIDs(data []byte) (versionIDs []string) {
	tmpData := allVersionsPattern.FindSubmatch(data)[1]
	allMatches := allVersionsLinePattern.FindAllSubmatch(tmpData, -1)
	for _, match := range allMatches {
		versionIDs = append(versionIDs, string(match[1]))
	}
	return versionIDs
}

func (c *ChangeHtml) Response(f *proxy.Flow) {
	//fmt.Println("----- On Response ----")
	//fmt.Println(f.Request.URL.String())
	//if strings.Contains(f.Request.URL.String(), targetUrlSubstr) {
	//	atomic.AddInt64(&counter, 1)
	//	responseBodyClone := bytes.Clone(f.Response.Body)
	//	responseBodyDecoded, err := GzipDecode(responseBodyClone)
	//	if err != nil {
	//		fmt.Println("Failed to decode the response body!")
	//		fmt.Println("ERROR:", err)
	//	}
	//	// write response body to file
	//	err = os.WriteFile("response"+strconv.Itoa(int(counter))+".xml", responseBodyDecoded, 0744)
	//	if err != nil {
	//		fmt.Println("[ERROR]", err)
	//	}
	//}
}

func Start(listenAddress string) {
	if listenAddress == "" {
		listenAddress = ":8080"
	}
	opts := &proxy.Options{
		Addr:              listenAddress,
		StreamLargeBodies: 1024 * 1024 * 5,
	}

	p, err := proxy.NewProxy(opts)
	if err != nil {
		log.Fatal(err)
	}

	// Add on
	//p.AddAddon(&AddHeader{})
	p.AddAddon(&ChangeHtml{})

	log.Fatal(p.Start())
}

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

const Version = "v0.1.0"

var Name = filepath.Base(os.Args[0])
var GitHash string

var done = make(chan bool)

var (
	V     = flag.Bool("v", false, "show version")
	H     = flag.Bool("h", false, "show help and exit")

	S     = flag.Bool("s", false, "do not set system proxy")
	C     = flag.Bool("c", false, "cleanup and exit. (e.g. turn off proxy)")
	PS    = flag.Bool("ps", false, "show current system proxy status")

	START = flag.Int("start", 0, "versionIDs index range [start, end)")
	END   = flag.Int("end", math.MaxInt, "versionIDs index range [start, end)")
)

var systemProxy = sysproxy.New()
const host, port = "127.0.0.1", 8080

func turnOnProxy() {
	fmt.Println("Turn on system proxy...")
	systemProxy.On(host, port)
}

func turnOffProxy() {
	fmt.Println("Turn off system proxy...")
	systemProxy.Off(host, port)
}

func showProxy() {
	systemProxy.Show()
}

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", Name)
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "options\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		return
	}

	if *H {
		flag.Usage()
		return
	}

	if *V {
		fmt.Printf("%s version %s %s\n", Name, Version, GitHash)
		return
	}

	if *PS {
		showProxy()
		return
	}

	if *C {
		turnOffProxy()
		return
	}

	if !*S {
		turnOnProxy()
		// turn off proxy upon normal quitting
		defer func() {
			turnOffProxy()
		}()
	}

	// do cleanup upon CTRL-C or other system signal
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\ncleanup...")
		if !*S {
			turnOffProxy()
		}
		os.Exit(1)
	}()

	// start server
	defer mu.Unlock()
	go func() {
		Start(":8080")
	}()
	// wait until finish
	<-done

	if !*S {
		fmt.Println("Turn off system proxy...")
		systemProxy.Off(host, port)
	}

	//TODO:
	//// Let user select one to download
	//var input string
	//for {
	//	fmt.Printf("Enter index number to download, or enter -1 to exit: ")
	//	_, err := fmt.Scanln(&input)
	//	if err != nil {
	//		fmt.Println(err)
	//		continue
	//	}
	//	selectedIndex, err := strconv.Atoi(input)
	//	if err != nil {
	//		fmt.Println("Invalid input.")
	//		continue
	//	}
	//
	//	if selectedIndex < 0 {
	//		break
	//	}
	//	if selectedIndex >= len(historyVersions) {
	//		fmt.Println("Invalid input: index out of range.")
	//		continue
	//	}
	//
	//	selectedVersion := historyVersions[selectedIndex]
	//	filename, err := Download(selectedVersion, clientUserAgent)
	//	if err != nil {
	//		continue
	//	}
	//	fmt.Printf("File [%s] saved.\n", filename)
	//	break
	//}

	// Exit when click Enter
	fmt.Printf("Press [Enter] to quit: ")
	fmt.Scanln()
}

func Download(app *AppInfo, userAgent string) (filename string, err error) {
	filename = fmt.Sprintf("%s %s.ipa", app.BundleDisplayName, app.BundleShortVersionString)
	fmt.Printf("Direct link: %s\n", app.URL)
	fmt.Printf("Downloading %s %s (%s) to file [%s]...\n", app.BundleDisplayName, app.BundleShortVersionString, app.SoftwareVersionExternalIdentifier, filename)
	// TODO:
	err = downloadFile(app.URL, filename, "")
	if err != nil {
		fmt.Println(err)
		return filename, err
	}
	return filename, nil
}


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
	fmt.Printf("\r%s", strings.Repeat(" ", 35))
	s := fmt.Sprintf("Progress: %.1f%%", float64(wc.Count)/float64(wc.Total)*100)
	fmt.Printf("\r%s", s)
	if wc.Count == wc.Total {
		fmt.Println()
	}
}

var defaultUserAgent = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.61 Mobile Safari/537.36"
func downloadFile(url string, filepath string, userAgent string) (err error) {
	if userAgent == "" {
		userAgent = defaultUserAgent
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
