package ipaversion

import (
	"fmt"
)

// AppInfo includes all fields from DownloadItemResult, and
// extracts most commonly used fields from metadata for easy access.
type AppInfo struct {
	*DownloadItemResult
	BundleDisplayName					string		// ex. Opera
	BundleShortVersionString			string		// ex. 3.0.4
	SoftwareVersionExternalIdentifier	uint64		// ex. 842552350
	SoftwareVersionExternalIdentifiers	[]uint64	// ex. [842023522, 842552350, 842626028]
	ItemName							string		// ex. Opera: 快速 &amp; 安全
	ArtistName							string		// ex. Opera Software AS
}

func GetAppInfo(data []byte) (*AppInfo, error) {
	item, err := GetDownloadItemResult(data)
	if err != nil {
		return nil, err
	}
	metadata := item.Metadata
	tmpIDs := metadata["softwareVersionExternalIdentifiers"].([]interface{})
	var versionIDs []uint64
	for _, tmpID := range tmpIDs {
		versionIDs = append(versionIDs, tmpID.(uint64))
	}
	appInfo := &AppInfo{
		DownloadItemResult:                	item,
		BundleDisplayName:                 	metadata["bundleDisplayName"].(string),
		BundleShortVersionString:          	metadata["bundleShortVersionString"].(string),
		SoftwareVersionExternalIdentifier: 	metadata["softwareVersionExternalIdentifier"].(uint64),
		SoftwareVersionExternalIdentifiers:	versionIDs,
		ItemName:                          	metadata["itemName"].(string),
		ArtistName:                        	metadata["artistName"].(string),
	}
	return appInfo, nil
}

func DownloadApp(app *AppInfo, userAgent string) (filename string, err error) {
	filename = fmt.Sprintf("%s %s.ipa", app.BundleDisplayName, app.BundleShortVersionString)
	fmt.Printf("Direct link: %s\n", app.URL)
	fmt.Printf("Downloading %s %s (%s) to file [%s]...\n", app.BundleDisplayName, app.BundleShortVersionString, app.SoftwareVersionExternalIdentifier, filename)
	// TODO:
	err = DownloadFile(app.URL, filename, userAgent)
	if err != nil {
		fmt.Println(err)
		return filename, err
	}
	return filename, nil
}
