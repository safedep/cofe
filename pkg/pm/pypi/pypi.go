package pypi

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	hm "github.com/safedep/cofe/pkg/pm/http_manager"
	"github.com/safedep/vet/pkg/common/logger"
)

var (
	PkgNotFound = errors.New("Pkg not found on the index server")
)

type PypiPackageManager struct {
	// pypiUrl string
	httpm *hm.HTTPClientManager
}

func NewPrivatePypiPackageManager(indexUrls []string) *PypiPackageManager {
	httm := hm.NewHTTPClientManager()
	for _, pypiUrl := range indexUrls {
		httm.AddURL(pypiUrl)
	}
	return &PypiPackageManager{httpm: httm}
}

func NewPypiPackageManager(indexUrls []string) *PypiPackageManager {
	return NewPrivatePypiPackageManager(indexUrls)
}

func (s *PypiPackageManager) DownloadAndGetPackageInfo(directory, packageName, version string) (map[string]interface{}, string, error) {

	clients := s.httpm.GetAllBaseURLs()
	if len(clients) == 0 {
		return nil, "", fmt.Errorf("No http clients found to get package info")
	}

	for _, client := range clients {
		data, extractDir, err := s.downloadPackage(client, packageName, directory, version)
		if err != nil {
			continue
		} else {
			return data, extractDir, nil
		}
	}
	return nil, "", fmt.Errorf("Unpexted Error. Error from all pypi servers")
}

func (s *PypiPackageManager) downloadPackage(client *hm.PypiHttpClient, packageName, directory, version string) (map[string]interface{}, string, error) {
	var data map[string]interface{}
	var err error
	data, err = s.getPackageInfo(client, packageName)
	if err != nil {
		return data, "", err
	}

	releases, ok := data["releases"].(map[string]interface{})
	if !ok {
		return data, "", fmt.Errorf("Invalid 'releases' data in package info data length %d", len(data))
	}

	latestVersion := ""
	for v, _ := range releases {
		latestVersion = v
		break
	}

	if version == "" || version == "0.0.0" {
		v, ok := data["info"].(map[string]interface{})["version"]
		if ok {
			version = v.(string)
		} else if latestVersion != "" {
			version = latestVersion
		} else {
			return data, "", fmt.Errorf("Error while extracting version %s", packageName)
		}
	}

	files, ok := releases[version].([]interface{})
	if !ok {
		return data, "", fmt.Errorf("Version %s for package %s doesn't exist", version, packageName)
	}

	var url, fileExtension string

	for _, file := range files {
		fileMap, ok := file.(map[string]interface{})
		if !ok {
			continue
		}

		filename, ok := fileMap["filename"].(string)
		if !ok {
			continue
		}

		if strings.HasSuffix(filename, ".tar.gz") {
			url, fileExtension = fileMap["url"].(string), ".tar.gz"
			break
		} else if strings.HasSuffix(filename, ".egg") || strings.HasSuffix(filename, ".whl") || strings.HasSuffix(filename, ".zip") {
			url, fileExtension = fileMap["url"].(string), filepath.Ext(filename)
			break
		}
	}

	if url == "" || fileExtension == "" {
		return data, "", errors.New(fmt.Sprintf("Compressed file for %s does not exist on PyPI", packageName))
	}

	zippath := filepath.Join(directory, packageName+fileExtension)
	unzippedpath := strings.TrimSuffix(zippath, fileExtension)

	err = s.downloadCompressed(url, zippath, unzippedpath)
	if err != nil {
		return data, "", err
	}

	return data, unzippedpath, nil
}

func (s *PypiPackageManager) getPackageInfo(client *hm.PypiHttpClient, name string) (map[string]interface{}, error) {

	var data map[string]interface{}

	url, err := url.JoinPath(client.BaseUrl, name, "json")
	if err != nil {
		return nil, err
	}
	logger.Debugf("Retrieving PyPI package metadata from %s\n", url)

	response, err := client.Get(url)
	if err != nil {
		logger.Debugf("Error while retrieving package %s\n", err)
		return nil, nil
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Received status code: %d from PyPI", response.StatusCode)
	}

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	if message, ok := data["message"].(string); ok {
		return nil, fmt.Errorf("Error retrieving package: %s", message)
	}

	return data, nil
}

func (s *PypiPackageManager) downloadCompressed(url, archivePath, targetPath string) error {
	logger.Debugf("Downloading package archive from %s into %s\n", url, targetPath)

	response, err := s.httpm.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	logger.Debugf("Extracting archive %s to directory %s\n", archivePath, targetPath)

	err = s.safeExtract(archivePath, targetPath)
	if err != nil {
		logger.Debugf("Error extracting the file: %v\n", err)
		return err
	}

	logger.Debugf("Successfully extracted files to %s\n", targetPath)

	logger.Debugf("Removing temporary archive file %s\n", archivePath)
	err = os.Remove(archivePath)
	if err != nil {
		logger.Debugf("Error removing temporary archive file: %v\n", err)
		return err
	}

	return nil
}

func (s *PypiPackageManager) safeExtract(sourceArchive, targetDirectory string) error {
	if strings.HasSuffix(sourceArchive, ".tar.gz") || strings.HasSuffix(sourceArchive, ".tgz") {
		return s.extractTarGz(sourceArchive, targetDirectory)
	} else if strings.HasSuffix(sourceArchive, ".zip") || strings.HasSuffix(sourceArchive, ".whl") {
		return s.extractZip(sourceArchive, targetDirectory)
	} else {
		return fmt.Errorf("unsupported archive extension: %s", sourceArchive)
	}
}

func (s *PypiPackageManager) extractTarGz(sourceArchive, targetDirectory string) error {
	file, err := os.Open(sourceArchive)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDirectory, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err := os.MkdirAll(targetPath, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
		case tar.TypeReg:
			file, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(file, tr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *PypiPackageManager) extractZip(sourceArchive, targetDirectory string) error {
	r, err := zip.OpenReader(sourceArchive)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		filePath := filepath.Join(targetDirectory, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
		} else {
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				return err
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}

			dstFile, err := os.Create(filePath)
			if err != nil {
				rc.Close()
				return err
			}

			_, err = io.Copy(dstFile, rc)
			rc.Close()
			dstFile.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
