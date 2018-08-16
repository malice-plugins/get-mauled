package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/malice-plugins/go-plugin-utils/utils"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	// Version stores the plugin's version
	Version string

	// BuildTime stores the plugin's build time
	BuildTime string
)

// Copy copies a file from src to dst
func Copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

// FlattenDir copies and flattens a whole directory recursively
func FlattenDir(src string, dst string) error {
	var err error
	var fds []os.FileInfo

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = FlattenDir(srcfp, dst); err != nil {
				fmt.Println(err)
			}
		} else {
			// err := os.Rename(srcfp, dstfp)
			_, err := Copy(srcfp, dstfp)
			if err != nil {
				return errors.Wrap(err, "failed to copy file")
			}
		}
	}
	return nil
}

func downloadFromURL(url string, tmpfile *os.File) error {
	// Download file
	log.Info("Downloading file: ", url)
	response, err := http.Get(url)
	if err != nil {
		return errors.Wrapf(err, "downloading %s failed", url)
	}
	defer response.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	if _, err := tmpfile.Write(buf.Bytes()); err != nil {
		log.Fatal(err)
	}
	if err != nil {
		return errors.Wrapf(err, "writing %s failed", tmpfile.Name())
	}

	return nil
}

func getPassword(fileName string) string {
	base := strings.TrimSuffix(fileName, path.Ext(fileName))
	decoded, _ := base64.StdEncoding.DecodeString("aW5mZWN0ZWQ2NjY=")
	return string(decoded) + base[len(base)-1:]
}

func unzipWithPassword(ctx context.Context, path, password, outputFolder string) (string, error) {
	var c *exec.Cmd

	args := []string{"x", path, fmt.Sprintf("-p%s", password), fmt.Sprintf("-o%s", outputFolder)}

	if ctx != nil {
		c = exec.CommandContext(ctx, "7z", args...)
	} else {
		c = exec.Command("7z", args...)
	}

	log.Debug("running command: ", c.Args)

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

func downloadAndUnzip(ctx context.Context, url, password, output string) error {

	tmpfile, err := ioutil.TempFile("", "getmauled")
	if err != nil {
		return errors.Wrap(err, "failed to create tmp file")
	}
	defer os.Remove(tmpfile.Name()) // clean up

	err = downloadFromURL(url, tmpfile)
	if err != nil {
		return errors.Wrapf(err, "downloading %s failed", url)
	}

	if err := tmpfile.Close(); err != nil {
		return errors.Wrap(err, "failed to close tmp file")
	}

	tmpDir, err := ioutil.TempDir("", "getmauled")
	if err != nil {
		return errors.Wrap(err, "failed to create tmp directory")
	}
	defer os.RemoveAll(tmpDir)

	out, err := unzipWithPassword(ctx, tmpfile.Name(), password, tmpDir)
	if err != nil {
		return errors.Wrapf(err, "unzipping %s failed", url)
	}
	log.Debug(out)

	err = FlattenDir(tmpDir, output)

	return nil
}

func main() {

	cli.AppHelpTemplate = utils.AppHelpTemplate
	app := cli.NewApp()

	app.Name = "get-mauled"
	app.Author = "blacktop"
	app.Email = "https://github.com/blacktop"
	app.Version = Version + ", BuildTime: " + BuildTime
	app.Compiled, _ = time.Parse("20060102", BuildTime)
	app.Usage = "Malice Demo Malware Downloader Plugin"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, V",
			Usage: "verbose output",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:      "download",
			Aliases:   []string{"d"},
			Usage:     "Download Malware",
			ArgsUsage: "URL to download",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:   "proxy, x",
					Usage:  "proxy settings for Malice webhook endpoint",
					EnvVar: "MALICE_PROXY",
				},
				cli.IntFlag{
					Name:   "timeout",
					Value:  60,
					Usage:  "malice plugin timeout (in seconds)",
					EnvVar: "MALICE_TIMEOUT",
				},
				cli.StringFlag{
					Name:   "password, p",
					Value:  "infected",
					Usage:  "password of malware zip",
					EnvVar: "MALICE_ZIP_PASSWORD",
				},
				cli.StringFlag{
					Name:   "output, o",
					Value:  "/malware",
					Usage:  "set output directory",
					EnvVar: "MALICE_OUTPUT_DIRECTORY",
				},
			},
			Action: func(c *cli.Context) error {

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Int("timeout"))*time.Second)
				defer cancel()

				urls := []string{
					"https://github.com/ytisf/theZoo/raw/master/malwares/Binaries/Duqu2/Duqu2.zip",
					"https://github.com/fabrimagic72/malware-samples/raw/master/Ransomware/Wannacry/smb-0e89k3id.zip",
					"https://github.com/fabrimagic72/malware-samples/raw/master/Ransomware/NotPetya/027cc450ef5f8c5f653329641ec1fed91f694e0d229928963b30f6b0d7d3a745-20170707033827.zip",
				}

				if c.Args().Present() {
					urls = []string{c.Args().First()}
				}

				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}

				if _, err := os.Stat(c.String("output")); os.IsNotExist(err) {
					return errors.Wrapf(err, "directory %s doesn't exist", c.String("output"))
				}

				for _, url := range urls {
					err := downloadAndUnzip(ctx, url, c.String("password"), c.String("output"))
					if err != nil {
						log.Fatal(err)
					}
				}

				// } else {
				// 	log.Fatal(fmt.Errorf("please supply a URL to download"))
				// }
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	utils.Assert(err)
}
