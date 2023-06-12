package util

import (
	"fmt"
	"regexp"
)

type AppInfo struct {
	BundleDisplayName string					// ex. Opera
	BundleShortVersionString string				// ex. 3.0.4
	SoftwareVersionExternalIdentifier string	// ex. 842552350
	ItemName string			// ex. Opera: 快速 &amp; 安全
	ArtistName string		// ex. Opera Software AS
	URL string				// ex. https://iosapps.itunes.apple.com/itunes-assets/../xx/yy/zz.signed.dpkg.ipa?accessKey=xxx
	ArtworkURL string		// ex. https://is4-ssl.mzstatic.com/image/thumb/Purple122/.../AppIcon-xxx.png/600x600bb.jpg
}

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

func DownloadApp(app *AppInfo, userAgent string) (filename string, err error) {
	filename = fmt.Sprintf("%s %s.ipa", app.BundleDisplayName, app.BundleShortVersionString)
	fmt.Printf("Direct link: %s\n", app.URL)
	fmt.Printf("Downloading %s %s (%s) to file [%s]...\n", app.BundleDisplayName, app.BundleShortVersionString, app.SoftwareVersionExternalIdentifier, filename)
	// TODO:
	err = DownloadFile(app.URL, filename, "")
	if err != nil {
		fmt.Println(err)
		return filename, err
	}
	return filename, nil
}
