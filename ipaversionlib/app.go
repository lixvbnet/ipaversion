package ipaversion

import (
	"fmt"
	"os"
)

// AppInfo includes all fields from DownloadItemResult, and
// extracts most commonly used fields from metadata for easy access.
type AppInfo struct {
	*DownloadItemResult
	Data								[]byte		// the original response data (gzip decoded)
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
		Data:								data,
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

func DownloadApp(app *AppInfo, userAgent string, overwrite bool) (filename string, exists bool, err error) {
	filename = fmt.Sprintf("%s %s.ipa", app.BundleDisplayName, app.BundleShortVersionString)
	exists = false
	if !overwrite {
		exists, err = fileExists(filename)
		if err != nil {
			return filename, exists, err
		}
		if exists {
			return filename, exists, nil
		}
	}

	//fmt.Printf("Direct link: %s\n", app.URL)
	fmt.Printf("Downloading %s %s (%v) to file [%s]...\n", app.BundleDisplayName, app.BundleShortVersionString, app.SoftwareVersionExternalIdentifier, filename)
	tmpFile := fmt.Sprintf("%s.downloading", filename)
	// download raw ipa to tmp
	err = DownloadFile(app.URL, tmpFile, userAgent)
	if err != nil {
		return filename, exists, err
	}
	// apply patches
	fmt.Println("Apply patches...")
	err = ApplyPatches(app.Metadata, tmpFile, filename)
	if err != nil {
		return filename, exists, fmt.Errorf("Failed to apply patches: %v\n", err)
	}
	// replicate sinfs
	fmt.Println("Replicate sinfs...")
	err = ReplicateSinf(app.Sinfs, filename)
	if err != nil {
		return filename, exists, fmt.Errorf("Failed to replicate sinfs: %v\n", err)
	}
	fmt.Println("Remove tmp files...")
	err = os.Remove(tmpFile)	// NOT working!
	if err != nil {
		return filename, exists, fmt.Errorf("Failed to remove tmp file: %v\n", err)
	}
	return filename, exists, nil
}
