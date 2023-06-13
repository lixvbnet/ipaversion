package main

import (
	"flag"
	"fmt"
	"github.com/lixvbnet/ipaversion/ipaversionlib"
	"github.com/lixvbnet/sysproxy"
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
)

const Version = "v0.1.0"

var Name = filepath.Base(os.Args[0])
var GitHash string

var (
	V     = flag.Bool("v", false, "show version")
	H     = flag.Bool("h", false, "show help and exit")

	S     = flag.Bool("s", false, "do not set system proxy")
	C     = flag.Bool("c", false, "cleanup and exit. (e.g. turn off proxy)")
	PS    = flag.Bool("ps", false, "show current system proxy status")

	START = flag.Int("start", 0, "versionIDs index range [start, end)")
	END   = flag.Int("end", math.MaxInt, "versionIDs index range [start, end)")
)

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
		// do cleanup when error happens
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("[WARN] Recovered from main: %v\n", err)
				turnOffProxy()
			}
		}()

		// do cleanup upon CTRL-C or other system interrupt signal
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			fmt.Println("\ncleanup...")
			turnOffProxy()
			os.Exit(1)
		}()
	}

	// lock when target url is matched; unlock when program exits
	var mu sync.Mutex
	defer mu.Unlock()
	var done = make(chan bool)

	// create proxy server
	listenAddr := fmt.Sprintf(":%d", port)
	opts := &proxy.Options{
		Addr:              listenAddr,
		StreamLargeBodies: 1024 * 1024 * 5,
	}
	p, err := proxy.NewProxy(opts)
	if err != nil {
		log.Fatal(err)
	}
	// Add on
	p.AddAddon(&ipaversion.Addon{
		Lock: &mu,
		Done: done,
		Start: *START,
		End: *END,
	})
	// start proxy server
	go func() {
		log.Fatal(p.Start())
	}()
	defer p.Close()

	// wait until finish
	ok := <-done
	if !ok {
		fmt.Printf("[ERROR] Error happened in the Addon.\n")
	}

	if !*S {
		turnOffProxy()
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
	_, _ = fmt.Scanln()
}


const host, port = "127.0.0.1", 8080
var systemProxy sysproxy.SysProxy

func turnOnProxy() {
	fmt.Println("Turn on system proxy...")
	if systemProxy == nil {
		systemProxy = sysproxy.New()
	}
	systemProxy.On(host, port)
}

func turnOffProxy() {
	fmt.Println("Turn off system proxy...")
	if systemProxy == nil {
		fmt.Println("Skip. As systemProxy is nil.")
		return
	}
	systemProxy.Off(host, port)
}

func showProxy() {
	if systemProxy == nil {
		systemProxy = sysproxy.New()
	}
	systemProxy.Show()
}
