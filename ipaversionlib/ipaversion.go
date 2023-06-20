package ipaversion

import (
	"bytes"
	"fmt"
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"golang.org/x/exp/maps"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const targetUrlSubstr = "-buy.itunes.apple.com/WebObjects/MZBuy.woa/wa/buyProduct"
const savedResponseDir = "savedResponse"

type Addon struct {
	proxy.BaseAddon
	Lock 				*sync.Mutex
	Done				chan<- bool

	Start				int
	End					int
	HistoryVersions 	[]*AppInfo		// This is query result. Do NOT change it from outside!
	ClientUserAgent		string			// User-Agent extracted from client request headers. Do NOT change it from outside!
	DumpResponses		bool			// Whether to dump replay responses to files

	counter				int64
}

type replayRequestInput struct {
	Method, Url                      string
	Header                           http.Header
	Body                             []byte
	LatestVersionID, TargetVersionID uint64
	Index                            int // The index in all versionIDs
	ActualIndex                      int // The actual index in the filtered query result HistoryVersions
	Client                           *http.Client
}

// replay the request with modified version ID
func (c *Addon) replayRequest(in *replayRequestInput, retry int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("[WARN] Recovered from replayRequest [%d] %v: %v.", in.Index, in.TargetVersionID, err)
			if retry > 0 {
				fmt.Println(" Retrying...")
				c.replayRequest(in, retry-1)
			} else {
				fmt.Println()
			}
		}
	}()
	atomic.AddInt64(&c.counter, 1)
	// clone the request for every versionID
	headerClone := maps.Clone(in.Header)
	bodyClone := bytes.Clone(in.Body)
	// modify the cloned request
	strOld := fmt.Sprintf("<string>%v</string>", in.LatestVersionID)
	strNew := fmt.Sprintf("<string>%v</string>", in.TargetVersionID)
	bodyClone = bytes.ReplaceAll(bodyClone, []byte(strOld), []byte(strNew))
	reqClone, _ := http.NewRequest(in.Method, in.Url, bytes.NewReader(bodyClone))
	reqClone.Header = headerClone

	resp, _ := in.Client.Do(reqClone)

	// handle the response
	resBodyDecoded, err := GzipDecodeReader(resp.Body)
	if err != nil {
		log.Panicln("Failed to decode the response body:", err)
	}
	err = resp.Body.Close()
	if err != nil {
		fmt.Println("[WARN] failed to close response body:", err)
	}
	// get versionID and versionStr
	appInfo, err := GetAppInfo(resBodyDecoded)
	if err != nil {
		log.Panicln("Failed to parse the app info:", err)
	}
	c.HistoryVersions = append(c.HistoryVersions, appInfo)
	fmt.Printf("(%d) [%d] %v %s\n", in.ActualIndex, in.Index, appInfo.SoftwareVersionExternalIdentifier, appInfo.BundleShortVersionString)
	// write the response body to file
	if c.DumpResponses {
		err = os.WriteFile(savedResponseDir+"/ReplayResponse"+strconv.Itoa(int(c.counter))+".xml", resBodyDecoded, 0744)
		if err != nil {
			fmt.Println("[ERROR]", err)
		}
	}
}

func (c *Addon) handleBuyRequest(f *proxy.Flow) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("[WARN] Recovered from handleBuyRequest [%s]: %v\n", f.Request.URL.String(), err)
			c.Done <- false
		}
	}()
	req := f.Request
	headerCopy := maps.Clone(f.Request.Header)
	bodyCopy := bytes.Clone(f.Request.Body)

	c.ClientUserAgent = headerCopy.Get("User-Agent")
	if c.ClientUserAgent == "" {
		c.ClientUserAgent = DefaultUserAgent
	}
	fmt.Println("User-Agent:", c.ClientUserAgent)

	//fmt.Println("Modifying request body..")
	//f.Request.Body = bytes.ReplaceAll(f.Request.Body, []byte("<string>852645599</string>"), []byte("<string>846783668</string>"))

	// clone the request & query all versionIDs
	client := &http.Client{}
	defer client.CloseIdleConnections()
	clonedRequest, err := CloneRequest(f.Request)
	if err != nil {
		log.Panicln("Failed to clone the request:", err)
	}
	res, err := client.Do(clonedRequest)
	if err != nil {
		log.Panicln("Failed to query all versionIDs:", err)
	}
	resBody, err := GzipDecodeReader(res.Body)
	if err != nil {
		log.Panicln("Failed to decode response body:", err)
	}
	err = res.Body.Close()
	if err != nil {
		fmt.Println("[WARN] failed to close res body:", err)
	}
	// get App info
	latestAppInfo, err := GetAppInfo(resBody)
	if err != nil {
		log.Panicln("Failed to parse latest app info:", err)
	}
	fmt.Println()
	fmt.Printf("[%s] (%s)\n", latestAppInfo.ItemName, latestAppInfo.ArtistName)
	fmt.Println("Latest version:\t\t", latestAppInfo.SoftwareVersionExternalIdentifier, latestAppInfo.BundleShortVersionString)
	// get all history versionIDs
	versionIDs := latestAppInfo.SoftwareVersionExternalIdentifiers
	fmt.Printf("History versionIDs:\t%v (Total: %d)\n", versionIDs, len(versionIDs))

	// calculate index range
	n := len(versionIDs)
	start, end := c.Start, c.End
	if start < 0 {					// if negative, then count from last
		start = n + start
	}
	if end < 0 {
		end = n + end
	}
	start, end = max(0, start), min(n, end)
	if start != 0 || end != n {						// check custom index range
		fmt.Printf("Index range: [%d, %d)\n", start, end)
		if start >= end {
			fmt.Println("Invalid range! Querying all versionIDs")
			start, end = 0, n
		}
	}

	// replay the request
	fmt.Println("Replaying the request...")
	fmt.Println()
	i := 0
	for index := start; index < end; index++ {
		versionID := versionIDs[index]
		c.replayRequest(&replayRequestInput{
			Method:          req.Method,
			Url:             req.URL.String(),
			Header:          headerCopy,
			Body:            bodyCopy,
			LatestVersionID: latestAppInfo.SoftwareVersionExternalIdentifier,
			TargetVersionID: versionID,
			Index:           index,
			ActualIndex:     i,
			Client:          client,
		}, 3)
		i++
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
	//	atomic.AddInt64(&c.counter, 1)
	//	responseBodyClone := bytes.Clone(f.Response.Body)
	//	responseBodyDecoded, err := GzipDecode(responseBodyClone)
	//	if err != nil {
	//		fmt.Println("Failed to decode the response body!")
	//		fmt.Println("ERROR:", err)
	//	}
	//	// write response body to file
	//	if c.DumpResponses {
	//		err = os.WriteFile(savedResponseDir+"/response"+strconv.Itoa(int(c.counter))+".xml", responseBodyDecoded, 0744)
	//		if err != nil {
	//			fmt.Println("[ERROR]", err)
	//		}
	//	}
	//}
}