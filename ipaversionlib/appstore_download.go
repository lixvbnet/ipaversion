package ipaversion

import (
	"fmt"
	"howett.net/plist"
)

type DownloadResult struct {
	FailureType		string						`plist:"failureType,omitempty"`
	CustomerMessage string						`plist:"customerMessage,omitempty"`
	Items			[]*DownloadItemResult		`plist:"songList,omitempty"`
}

type DownloadItemResult struct {
	MD5				string						`plist:"md5,omitempty"`
	URL				string						`plist:"URL,omitempty"`			// ex. https://iosapps.itunes.apple.com/itunes-assets/../xx/yy/zz.signed.dpkg.ipa?accessKey=xxx
	ArtworkURL		string						`plist:"artworkURL,omitempty"`	// ex. https://is4-ssl.mzstatic.com/image/thumb/Purple122/.../AppIcon-xxx.png/600x600bb.jpg
	Sinfs			[]*Sinf						`plist:"sinfs,omitempty"`
	Metadata		map[string]interface{}		`plist:"metadata,omitempty"`
}


func GetDownloadItemResult(data []byte) (*DownloadItemResult, error) {
	var result DownloadResult
	var err error
	_, err = plist.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	if result.FailureType != "" {
		return nil, fmt.Errorf("failureType: %s, customerMessage: %s", result.FailureType, result.CustomerMessage)
	}
	return result.Items[0], nil
}
