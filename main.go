package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/lixvbnet/ipaversion/util"
	"github.com/lixvbnet/sysproxy"
	"golang.org/x/exp/maps"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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

// All history versions
var historyVersions []*util.AppInfo
var clientUserAgent = util.DefaultUserAgent


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
		clonedRequest, err := util.CloneRequest(f.Request)
		if err != nil {
			fmt.Println("Failed to clone the request!")
			log.Fatal("ERROR:", err)
		}
		res, _ := client.Do(clonedRequest)
		resBody, err := util.GzipDecodeReader(res.Body)
		res.Body.Close()
		if err != nil {
			fmt.Println("Failed to decode response body!")
			log.Fatal("ERROR:", err)
		}
		// get App info
		latestAppInfo := util.GetAppInfo(resBody)
		fmt.Println()
		fmt.Printf("[%s] (%s)\n", latestAppInfo.ItemName, latestAppInfo.ArtistName)
		fmt.Println("Latest version:\t\t", latestAppInfo.SoftwareVersionExternalIdentifier, latestAppInfo.BundleShortVersionString)
		// get all history versionIDs
		versionIDs := util.GetAllAppVersionIDs(resBody)
		fmt.Println("History versionIDs:\t", versionIDs)

		// replay the request
		fmt.Println("Replaying the request...")
		fmt.Println()
		// clone the request for every versionID
		//targets := []string{"842552350", "842023522", "842626028", "842726114", "844540208"}
		targets := versionIDs
		start, end := util.Max(0, *START), util.Min(len(targets), *END)
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
			resBodyDecoded, err := util.GzipDecodeReader(resp.Body)
			resp.Body.Close()
			if err != nil {
				fmt.Println("Failed to decode the response body!")
				fmt.Println("ERROR:", err)
			}
			// get versionID and versionStr
			appInfo := util.GetAppInfo(resBodyDecoded)
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
	p.AddAddon(&ChangeHtml{})

	log.Fatal(p.Start())
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
	//	filename, err := util.DownloadApp(selectedVersion, clientUserAgent)
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
