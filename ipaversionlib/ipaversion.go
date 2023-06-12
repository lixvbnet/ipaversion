package ipaversion

import (
	"bytes"
	"fmt"
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"golang.org/x/exp/maps"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

const targetUrlSubstr = "-buy.itunes.apple.com/WebObjects/MZBuy.woa/wa/buyProduct"

var clientUserAgent = DefaultUserAgent
var counter int64 = 0


type Addon struct {
	proxy.BaseAddon
	Lock 				*sync.Mutex
	Done				chan<- bool

	Start				int
	End					int

	HistoryVersions 	[]*AppInfo		// This is query result. Do NOT change it from outside!
}

// replay the request with modified version ID
func (c *Addon) replayRequest(method, url string, header http.Header, body []byte, latestVersionID, targetVersionID string, i int, client *http.Client) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("[WARN] Recovered from replayRequest [%d] (%s): %v\n", i, targetVersionID, err)
		}
	}()
	atomic.AddInt64(&counter, 1)
	// clone the request for every versionID
	headerClone := maps.Clone(header)
	bodyClone := bytes.Clone(body)
	// modify the cloned request
	strOld := fmt.Sprintf("<string>%s</string>", latestVersionID)
	strNew := fmt.Sprintf("<string>%s</string>", targetVersionID)
	bodyClone = bytes.ReplaceAll(bodyClone, []byte(strOld), []byte(strNew))
	reqClone, _ := http.NewRequest(method, url, bytes.NewReader(bodyClone))
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
	c.HistoryVersions = append(c.HistoryVersions, appInfo)
	fmt.Printf("[%d] %s %s\n", i, appInfo.SoftwareVersionExternalIdentifier, appInfo.BundleShortVersionString)
	// write the response body to file
	//err = os.WriteFile("ReplayResponse"+strconv.Itoa(int(counter))+".xml", resBodyDecoded, 0744)
	//if err != nil {
	//	fmt.Println("[ERROR]", err)
	//}
}

func (c *Addon) handleBuyRequest(f *proxy.Flow) {
	req := f.Request

	headerCopy := maps.Clone(f.Request.Header)
	bodyCopy := bytes.Clone(f.Request.Body)

	clientUserAgent = headerCopy.Get("User-Agent")
	fmt.Println("User-Agent:", clientUserAgent)

	//fmt.Println("Modifying request body..")
	//f.Request.Body = bytes.ReplaceAll(f.Request.Body, []byte("<string>852645599</string>"), []byte("<string>846783668</string>"))

	// clone the request & query all versionIDs
	client := &http.Client{}
	defer client.CloseIdleConnections()
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
	n := len(versionIDs)
	start, end := c.Start, c.End
	if start < 0 {			// if negative, then count from last
		start = n + start
	}
	if end < 0 {
		end = n + start
	}
	start, end = max(0, c.Start), min(n, c.End)
	fmt.Printf("Index range: [%d, %d)\n", start, end)
	if start >= end {
		fmt.Println("Invalid range! Querying all versionIDs")
		start, end = 0, n
	}
	for i := start; i < end; i++ {
		versionID := versionIDs[i]
		c.replayRequest(req.Method, req.URL.String(), headerCopy, bodyCopy, latestAppInfo.SoftwareVersionExternalIdentifier, versionID, i, client)
	}

	fmt.Println("---------------------------------------")
	fmt.Println("Done!")
	c.Done <- true
}


func (c *Addon) Request(f *proxy.Flow) {
	req := f.Request
	url := req.URL
	//fmt.Println("----- (On Request) ----")
	//fmt.Println(url.String())

	if strings.Contains(url.String(), targetUrlSubstr) {
		// we only want to run the function code ONCE!
		if !c.Lock.TryLock() {
			return
		}

		c.handleBuyRequest(f)
	}
}

func (c *Addon) Response(f *proxy.Flow) {
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