package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const timeout = 60

// Version stores the plugin's version
var Version string

// BuildTime stores the plugin's build time
var BuildTime string

func downloadFromURL(dlURL string) (string, error) {

	u, err := url.Parse(dlURL)
	if err != nil {
		return "", errors.Wrapf(err, "failed parsing url: %s", dlURL)
	}
	fileName := filepath.Base(u.Path)

	// Check if files already exists
	if _, err := os.Stat(fileName); os.IsNotExist(err) {

		output, err := os.Create(fileName)
		if err != nil {
			return "", errors.Wrapf(err, "creating %s failed", fileName)
		}
		defer output.Close()

		response, err := http.Get(dlURL)
		if err != nil {
			return "", errors.Wrapf(err, "downloading %s failed", fileName)
		}
		defer response.Body.Close()

		_, err = io.Copy(output, response.Body)
		if err != nil {
			return "", errors.Wrapf(err, "writing %s failed", fileName)
		}

		return fileName, nil
	}

	return "", fmt.Errorf("file %s already exists", fileName)
}

func unzipWithPassword(ctx context.Context, path, password string) (string, error) {
	var c *exec.Cmd

	args := []string{"x", path, fmt.Sprintf("-p%s", password)}

	if ctx != nil {
		c = exec.CommandContext(ctx, "7z", args...)
	} else {
		c = exec.Command("7z", args...)
	}
	fmt.Println(c)
	output, err := c.Output()
	if err != nil {
		return string(output), errors.Wrap(err, "7zip extraction failed")
	}

	// check for exec context timeout
	if ctx != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", errors.Wrap(ctx.Err(), "command 7z x timed out")
		}
	}

	return string(output), nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	urls := []string{"GB", "FR", "ES", "DE", "CN", "CA", "ID", "US"}
	for _, u := range urls {
		url := "http://download.geonames.org/export/dump/" + u + ".zip"
		url = "https://github.com/ytisf/theZoo/raw/master/malwares/Binaries/Duqu2/Duqu2.zip"
		dlPath, err := downloadFromURL(url)
		if err != nil {
			log.Fatal(errors.Wrapf(err, "downloading %s failed", url))
		}
		out, err := unzipWithPassword(ctx, dlPath, "infected")
		if err != nil {
			log.Fatal(errors.Wrapf(err, "unzipping %s failed", url))
		}
		fmt.Println(out)
	}
}
