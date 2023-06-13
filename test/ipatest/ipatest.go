package main

import (
	"fmt"
	"github.com/lixvbnet/ipaversion/ipaversionlib"
	"howett.net/plist"
	"log"
	"os"
)

type downloadResult struct {
	FailureType     string                      `plist:"failureType,omitempty"`
	CustomerMessage string                      `plist:"customerMessage,omitempty"`
	Items           []*downloadItemResult       `plist:"songList,omitempty"`
}

type downloadItemResult struct {
	MD5             string                      `plist:"md5,omitempty"`
	URL             string                      `plist:"URL,omitempty"`
	ArtworkURL      string                      `plist:"artworkURL,omitempty"`
	Sinfs           []*ipaversion.Sinf          `plist:"sinfs,omitempty"`
	Metadata        map[string]interface{}      `plist:"metadata,omitempty"`
}

func main() {
	src := "Opera_3.src.ipa"
	dst := "Opera_3.ipa"

	data, _ := os.ReadFile("savedResponse/ReplayResponse3.xml")
	var appInfo downloadResult
	_, err := plist.Unmarshal(data, &appInfo)
	if err != nil {
		log.Fatal(err)
	}

	item := appInfo.Items[0]
	//sinfs := item.Sinfs

	fmt.Println("Apply patches...")
	err = ipaversion.ApplyPatches(item.Metadata, src, dst)
	if err != nil {
		log.Fatal("Failed to apply patches:", err)
	}
	fmt.Println("Replicate sinfs...")
	err = ipaversion.ReplicateSinf(item.Sinfs, dst)
	if err != nil {
		log.Fatal("Failed to replicate sinfs:", err)
	}
	fmt.Println("Success!")
}

