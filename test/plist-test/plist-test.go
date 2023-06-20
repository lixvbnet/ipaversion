package main

import (
	"fmt"
	ipaversion "github.com/lixvbnet/ipaversion/ipaversionlib"
	"log"
	"os"
)

func main() {
	data, _ := os.ReadFile("savedResponse/ReplayResponse1.xml")
	fmt.Println(len(data))
	//var downloadResult ipaversion.DownloadResult
	//_, err := plist.Unmarshal(data, &downloadResult)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Println(downloadResult.Items[0])

	appInfo, err := ipaversion.GetAppInfo(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(appInfo.Data))

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
}
