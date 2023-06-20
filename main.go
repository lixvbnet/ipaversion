package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/lixvbnet/ipaversion/ipaversionlib"
	"github.com/lixvbnet/sysproxy"
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unicode"
)

const Version = "v0.2.0"

var Name = filepath.Base(os.Args[0])
var GitHash string

var (
	V     = flag.Bool("v", false, "show version")
	H     = flag.Bool("h", false, "show help and exit")

	I     = flag.String("i", "", "read the input file and download ipa")

	PS    = flag.Bool("ps", false, "show current system proxy status")
	C     = flag.Bool("c", false, "cleanup and exit. (e.g. turn off proxy)")
	S     = flag.Bool("s", false, "do not set system proxy")

	DUMP  = flag.Bool("dump", false, "dump responses to files")

	START = flag.Int("start", 0, "versionIDs index range [start, end)")
	END   = flag.Int("end", math.MaxInt, "versionIDs index range [start, end)")
)

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", Name)
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "options\n")
		flag.PrintDefaults()
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "\ncommands\n")
		printAvailableCommands(flag.CommandLine.Output())
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

	if *I != "" {
		fmt.Printf("Reading response data from [%s]...\n", *I)
		data, err := os.ReadFile(*I)
		if err != nil {
			log.Fatal(err)
		}
		appInfo, err := ipaversion.GetAppInfo(data)
		if err != nil {
			log.Fatal(err)
		}
		// download app
		filename, exists, err := ipaversion.DownloadApp(appInfo, "", false)
		if err != nil {
			log.Fatal(err)
		}
		if exists {
			fmt.Printf("File [%s] already exists. [Skip]\n", filename)
		} else {
			fmt.Printf("File [%s] saved\n", filename)
		}
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
	addon := &ipaversion.Addon{
		Lock: &mu,
		Done: done,
		Start: *START,
		End: *END,
		DumpResponses: *DUMP,
	}
	p.AddAddon(addon)
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

	// Let user input an index number to download, or other commands
	handleInput(addon)

	// Exit when click Enter
	fmt.Printf("Press [Enter] to quit: ")
	_, _ = fmt.Scanln()
}


var availableCommands = [][2]string{
	{"?", 				"show help message"},
	{"exit", 			"exit"},
	{"d|dump <index>", 	"dump response data (of the given index) to file"},
}

func printAvailableCommands(w io.Writer) {
	//_, _ = fmt.Fprintln(w, "Available commands:")
	for _, cmd := range availableCommands {
		_, _ = fmt.Fprintf(w, "%-20s %s\n", cmd[0], cmd[1])
	}
}

func handleInput(addon *ipaversion.Addon) {
	historyVersions, clientUserAgent := addon.HistoryVersions, addon.ClientUserAgent
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Enter index number to download, or '?' for available commands: ")
		input, err := r.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}
		input = strings.TrimSpace(input)

		if unicode.IsDigit(rune(input[0])) {	// consider input is a number
			selectedIndex, err := strconv.Atoi(input)
			if err != nil {
				fmt.Println("Invalid input.")
				continue
			}
			if selectedIndex < 0 {
				break
			}
			if selectedIndex >= len(historyVersions) {
				fmt.Println("Invalid input: index out of range.")
				continue
			}
			selectedVersion := historyVersions[selectedIndex]
			filename, _, err := ipaversion.DownloadApp(selectedVersion, clientUserAgent, true)
			if err != nil {
				fmt.Println("[ERROR]", err)
				continue
			}
			fmt.Printf("File [%s] saved.\n", filename)
			break

		} else {	// input is a command
			arr := strings.Fields(input)
			command := arr[0]
			if command == "?" {
				printAvailableCommands(os.Stdout)
				fmt.Println()
				continue
			} else if command == "exit" {
				break
			} else if command == "d" || command == "dump" {
				if len(arr) < 2 {
					printAvailableCommands(os.Stdout)
					fmt.Println()
					continue
				}
				selectedIndex, err := strconv.Atoi(arr[1])
				if err != nil || selectedIndex < 0 || selectedIndex >= len(historyVersions) {
					fmt.Println("Invalid input or index out of range.")
					continue
				}
				selectedVersion := historyVersions[selectedIndex]
				filename := fmt.Sprintf("%s_%s_%v.xml", selectedVersion.BundleDisplayName, selectedVersion.BundleShortVersionString, selectedVersion.SoftwareVersionExternalIdentifier)
				fmt.Printf("Saving response data to [%s]...\n", filename)
				err = os.WriteFile(filename, selectedVersion.Data, 0744)
				if err != nil {
					fmt.Println("[ERROR]", err)
					continue
				}
				fmt.Println()
			} else {
				fmt.Println("Unsupported command:", command)
				continue
			}
		}
	}
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
