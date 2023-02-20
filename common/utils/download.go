package utils

import (
	"github.com/scroll-tech/go-ethereum/log"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadToDir downloads the file and store it into dir, will do nothing if file exists.
// Return the downloaded file path.
func DownloadToDir(dir, url string) (string, error) {
	fileName := filepath.Base(url)
	path := filepath.Join(dir, fileName)

	exists, err := PathExists(path)
	if err != nil {
		return "", err
	}
	if exists {
		return path, nil
	}

	return path, download(path, url)
}

// DownloadFile downloads file and save in the given path.
// If file exists, it does nothing.
func DownloadFile(path, url string) error {
	exists, err := PathExists(path)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return download(path, url)
}

// PathExists checks if path exists.
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func download(path, url string) error {
	log.Info("Download from %s ......", url)
	// download file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// create new file
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	// store file into path
	_, err = io.Copy(f, resp.Body)
	return err
}
