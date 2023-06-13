package ipaversion

// Code copied from majd/ipatool, with minor modifications.

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"howett.net/plist"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ApplyPatches creates iTunesMetadata.plist and appends to dst file.
func ApplyPatches(metadata map[string]interface{}, src, dst string) error {
	srcZip, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip reader: %w", err)
	}
	defer srcZip.Close()

	//dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0644)
	dstFile, err := os.Create(dst)			// overwrite if existing
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	dstZip := zip.NewWriter(dstFile)
	defer dstZip.Close()

	err = replicateZip(srcZip, dstZip)
	if err != nil {
		return fmt.Errorf("failed to replicate zip: %w", err)
	}

	err = writeMetadata(metadata, dstZip)
	if err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func writeMetadata(metadata map[string]interface{}, zip *zip.Writer) error {
	//metadata["apple-id"] = appleID	// not required
	//metadata["userName"] = appleID	// not required

	metadataFile, err := zip.Create("iTunesMetadata.plist")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	//data, err := plist.Marshal(metadata, plist.BinaryFormat)
	//data, err := plist.Marshal(metadata, plist.XMLFormat)
	data, err := plist.MarshalIndent(metadata, plist.XMLFormat, "\t")	// more readable
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	_, err = metadataFile.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}


type Sinf struct {
	ID   int64  `plist:"id,omitempty"`
	Data []byte `plist:"sinf,omitempty"`
}

// ReplicateSinf writes sinfs to Payload/{NAME}.app/SC_Info/{NAME}.sinf
func ReplicateSinf(sinfs []*Sinf, packagePath string) error {
	zipReader, err := zip.OpenReader(packagePath)
	if err != nil {
		return errors.New("failed to open zip reader")
	}
	defer zipReader.Close()

	tmpPath := fmt.Sprintf("%s.tmp", packagePath)
	//tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY, 0644)
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	zipWriter := zip.NewWriter(tmpFile)
	defer zipWriter.Close()

	err = replicateZip(zipReader, zipWriter)
	if err != nil {
		return fmt.Errorf("failed to replicate zip: %w", err)
	}

	bundleName, err := readBundleName(zipReader)
	if err != nil {
		return fmt.Errorf("failed to read bundle name: %w", err)
	}

	manifest, err := readManifestPlist(zipReader)
	if err != nil {
		return fmt.Errorf("failed to read manifest plist: %w", err)
	}

	info, err := readInfoPlist(zipReader)
	if err != nil {
		return fmt.Errorf("failed to read info plist: %w", err)
	}

	if manifest != nil {
		err = replicateSinfFromManifest(*manifest, zipWriter, sinfs, bundleName)
	} else {
		err = replicateSinfFromInfo(*info, zipWriter, sinfs, bundleName)
	}
	if err != nil {
		return fmt.Errorf("failed to replicate sinf: %w", err)
	}

	// must explicitly close the files before remove/rename
	zipReader.Close()
	zipWriter.Close()
	tmpFile.Close()

	err = os.Remove(packagePath)
	if err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	err = os.Rename(tmpPath, packagePath)
	if err != nil {
		return fmt.Errorf("failed to rename the tmp file: %w", err)
	}

	return nil
}

type packageManifest struct {
	SinfPaths []string `plist:"SinfPaths,omitempty"`
}

type packageInfo struct {
	BundleExecutable string `plist:"CFBundleExecutable,omitempty"`
}

func replicateSinfFromManifest(manifest packageManifest, zip *zip.Writer, sinfs []*Sinf, bundleName string) error {
	zipped, err := Zip(sinfs, manifest.SinfPaths)
	if err != nil {
		return fmt.Errorf("failed to zip sinfs: %w", err)
	}

	for _, pair := range zipped {
		sp := fmt.Sprintf("Payload/%s.app/%s", bundleName, pair.Second)

		file, err := zip.Create(sp)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		_, err = file.Write(pair.First.Data)
		if err != nil {
			return fmt.Errorf("failed to write data: %w", err)
		}
	}

	return nil
}

func replicateSinfFromInfo(info packageInfo, zip *zip.Writer, sinfs []*Sinf, bundleName string) error {
	sp := fmt.Sprintf("Payload/%s.app/SC_Info/%s.sinf", bundleName, info.BundleExecutable)

	file, err := zip.Create(sp)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	_, err = file.Write(sinfs[0].Data)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

func replicateZip(src *zip.ReadCloser, dst *zip.Writer) error {
	for _, file := range src.File {
		srcFile, err := file.OpenRaw()
		if err != nil {
			return fmt.Errorf("failed to open raw file: %w", err)
		}

		header := file.FileHeader
		dstFile, err := dst.CreateRaw(&header)

		if err != nil {
			return fmt.Errorf("failed to create raw file: %w", err)
		}

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
	}

	return nil
}

func readInfoPlist(reader *zip.ReadCloser) (*packageInfo, error) {
	for _, file := range reader.File {
		if strings.Contains(file.Name, ".app/Info.plist") {
			src, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file: %w", err)
			}

			data := new(bytes.Buffer)
			_, err = io.Copy(data, src)

			if err != nil {
				return nil, fmt.Errorf("failed to copy data: %w", err)
			}

			var info packageInfo
			_, err = plist.Unmarshal(data.Bytes(), &info)

			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal data: %w", err)
			}

			return &info, nil
		}
	}

	return nil, nil
}

func readManifestPlist(reader *zip.ReadCloser) (*packageManifest, error) {
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".app/SC_Info/Manifest.plist") {
			src, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file: %w", err)
			}

			data := new(bytes.Buffer)
			_, err = io.Copy(data, src)

			if err != nil {
				return nil, fmt.Errorf("failed to copy data: %w", err)
			}

			var manifest packageManifest

			_, err = plist.Unmarshal(data.Bytes(), &manifest)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal data: %w", err)
			}

			return &manifest, nil
		}
	}

	return nil, nil
}

func readBundleName(reader *zip.ReadCloser) (string, error) {
	var bundleName string

	for _, file := range reader.File {
		if strings.Contains(file.Name, ".app/Info.plist") && !strings.Contains(file.Name, "/Watch/") {
			bundleName = filepath.Base(strings.TrimSuffix(file.Name, ".app/Info.plist"))

			break
		}
	}

	if bundleName == "" {
		return "", errors.New("could not read bundle name")
	}

	return bundleName, nil
}

type Pair[T, U any] struct {
	First  T
	Second U
}

func Zip[T, U any](ts []T, us []U) ([]Pair[T, U], error) {
	if len(ts) != len(us) {
		return nil, errors.New("slices have different lengths")
	}

	pairs := make([]Pair[T, U], len(ts))
	for i := 0; i < len(ts); i++ {
		pairs[i] = Pair[T, U]{
			First:  ts[i],
			Second: us[i],
		}
	}

	return pairs, nil
}
