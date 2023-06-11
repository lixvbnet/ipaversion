package main

import (
	"fmt"
	"howett.net/plist"
	"log"
	"os"
)

type Sinf struct {
	ID   int64  `plist:"id,omitempty"`
	Data []byte `plist:"sinf,omitempty"`
}

type downloadItemResult struct {
	HashMD5  string                 `plist:"md5,omitempty"`
	URL      string                 `plist:"URL,omitempty"`
	Sinfs    []Sinf                 `plist:"sinfs,omitempty"`
	Metadata map[string]interface{} `plist:"metadata,omitempty"`
}

type downloadResult struct {
	FailureType     string               `plist:"failureType,omitempty"`
	CustomerMessage string               `plist:"customerMessage,omitempty"`
	Items           []downloadItemResult `plist:"songList,omitempty"`
}

func main() {
	data, _ := os.ReadFile("savedResponse/ReplayResponse3.xml")
	fmt.Println(len(data))
	var appInfo downloadResult
	_, err := plist.Unmarshal(data, &appInfo)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(appInfo)
}
