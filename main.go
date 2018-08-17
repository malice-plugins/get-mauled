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
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/malice-plugins/go-plugin-utils/utils"
	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"

	"github.com/urfave/cli"
)

const (
	contagioDumpURL   = "https://www.dropbox.com/sh/i6ed6v32x0fp94z/AAAQvOsOvbWrOs8T3_ZTXqQya"
	theZooURL         = "https://github.com/ytisf/theZoo.git"
	malwareSamplesURL = "https://github.com/fabrimagic72/malware-samples.git"
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
			log.Debugf("wrote file to output folder: %s", dstfp)
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

func getPassword(fn string) string {
	b := strings.TrimSuffix(fn, path.Ext(fn))
	d, _ := base64.StdEncoding.DecodeString("aW5mZWN0ZWQ2NjY=")
	return string(d) + b[len(b)-1:]
}

func unzip(ctx context.Context, path, password, outputFolder string) (string, error) {
	var c *exec.Cmd
	var args []string

	if len(password) > 0 {
		args = []string{"x", path, fmt.Sprintf("-p%s", password), fmt.Sprintf("-o%s", outputFolder), "-y"}
	} else {
		args = []string{"x", path, fmt.Sprintf("-o%s", outputFolder), "-y"}
	}

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

	out, err := unzip(ctx, tmpfile.Name(), password, tmpDir)
	if err != nil {
		return errors.Wrapf(err, "unzipping %s failed", url)
	}
	log.Debug(out)

	err = FlattenDir(tmpDir, output)

	return nil
}

func gitCloneAndUnzip(ctx context.Context, url, output string) error {

	tmpDir, err := ioutil.TempDir("", "clone")
	if err != nil {
		return errors.Wrap(err, "failed to create tmp directory")
	}
	defer os.RemoveAll(tmpDir) // clean up

	// Clones the repository into the given tmpDir, just as a normal git clone does
	_, err = git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to clone from URL %s", url)
	}

	zipFiles, _ := filepath.Glob(tmpDir + "/*/*.zip")
	sevenZipFiles, _ := filepath.Glob(tmpDir + "/*/*.7z")
	zipFiles = append(zipFiles, sevenZipFiles...)

	for _, zipFile := range zipFiles {
		fmt.Println(zipFile)
		// out, err := unzip(ctx, zipFile, "infected", tmpDir)
		// if err != nil {
		// 	return errors.Wrapf(err, "unzipping %s failed", url)
		// }
		// log.Debug(out)
	}

	// err = FlattenDir(tmpDir, output)

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
	app.Usage = "Malice DEMO Malware Downloader Plugin"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, V",
			Usage: "verbose output",
		},
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
	}
	app.Commands = []cli.Command{
		{
			Name:      "download",
			Aliases:   []string{"d"},
			Usage:     "Download and Unzip Malware From URL",
			ArgsUsage: "url-to-download",
			Flags: []cli.Flag{

				cli.StringFlag{
					Name:   "password, p",
					Usage:  "password of malware zip",
					EnvVar: "MALICE_ZIP_PASSWORD",
				},
				cli.StringFlag{
					Name:   "output, o",
					Usage:  "set output directory",
					EnvVar: "MALICE_OUTPUT_DIRECTORY",
				},
			},
			Action: func(c *cli.Context) error {

				var err error
				var output string

				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.GlobalInt("timeout"))*time.Second)
				defer cancel()

				if len(c.String("output")) > 0 {
					output = c.String("output")
					if _, err = os.Stat(output); os.IsNotExist(err) {
						return errors.Wrapf(err, "directory %s doesn't exist", output)
					}
				} else {
					output, err = os.Getwd()
					if err != nil {
						return errors.Wrap(err, "unable to get current working directory")
					}
				}

				if c.Args().Present() {
					err = downloadAndUnzip(ctx, c.Args().First(), c.String("password"), output)
					if err != nil {
						log.Fatal(err)
					}

				} else {
					log.Fatal(fmt.Errorf("please supply a URL to download and unzip"))
				}
				return nil
			},
		},
		{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Gotta' Catch Em' All",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "output, o",
					Usage:  "set output directory",
					EnvVar: "MALICE_OUTPUT_DIRECTORY",
				},
			},
			Action: func(c *cli.Context) error {

				var err error
				var output string

				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.GlobalInt("timeout"))*time.Second)
				defer cancel()

				if len(c.String("output")) > 0 {
					output = c.String("output")
					if _, err = os.Stat(output); os.IsNotExist(err) {
						return errors.Wrapf(err, "directory %s doesn't exist", output)
					}
				} else {
					output, err = os.Getwd()
					if err != nil {
						return errors.Wrap(err, "unable to get current working directory")
					}
				}

				err = downloadAndUnzip(ctx, c.Args().First(), c.String("password"), output)
				if err != nil {
					log.Fatal(err)
				}

				return nil
			},
		},
		{
			Name:    "contagio",
			Aliases: []string{"c"},
			Usage:   "Download and Unzip contagiodump Malware",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "output, o",
					Usage:  "set output directory",
					EnvVar: "MALICE_OUTPUT_DIRECTORY",
				},
			},
			Action: func(c *cli.Context) error {

				var err error
				var output string

				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.GlobalInt("timeout"))*time.Second)
				defer cancel()

				if len(c.String("output")) > 0 {
					output = c.String("output")
					if _, err = os.Stat(output); os.IsNotExist(err) {
						return errors.Wrapf(err, "directory %s doesn't exist", output)
					}
				} else {
					output, err = os.Getwd()
					if err != nil {
						return errors.Wrap(err, "unable to get current working directory")
					}
				}

				err = gitCloneAndUnzip(ctx, malwareSamplesURL, output)
				if err != nil {
					log.Fatal(err)
				}

				return nil
			},
		},
		{
			Name:    "malware-samples",
			Aliases: []string{"m"},
			Usage:   "Download and Unzip Malware Samples",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "output, o",
					Usage:  "set output directory",
					EnvVar: "MALICE_OUTPUT_DIRECTORY",
				},
			},
			Action: func(c *cli.Context) error {

				var err error
				var output string

				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.GlobalInt("timeout"))*time.Second)
				defer cancel()

				if len(c.String("output")) > 0 {
					output = c.String("output")
					if _, err = os.Stat(output); os.IsNotExist(err) {
						return errors.Wrapf(err, "directory %s doesn't exist", output)
					}
				} else {
					output, err = os.Getwd()
					if err != nil {
						return errors.Wrap(err, "unable to get current working directory")
					}
				}

				tmpDir, err := ioutil.TempDir("", "clone")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp directory")
				}
				defer os.RemoveAll(tmpDir) // clean up

				// Clones the repository into the given tmpDir, just as a normal git clone does
				_, err = git.PlainClone(tmpDir, false, &git.CloneOptions{
					URL: malwareSamplesURL,
				})
				if err != nil {
					return errors.Wrapf(err, "failed to clone from URL %s", malwareSamplesURL)
				}

				zipFiles, _ := filepath.Glob(tmpDir + "/*/*.zip")
				sevenZipFiles, _ := filepath.Glob(tmpDir + "/*/*.7z")
				zipFiles = append(zipFiles, sevenZipFiles...)

				for _, zipFile := range zipFiles {
					fmt.Println(zipFile)
					out, _ := unzip(ctx, zipFile, "infected", tmpDir)
					// if err != nil {
					// 	return errors.Wrapf(err, "unzipping %s failed", zipFile)
					// }
					os.Remove(zipFile)
					log.Debug(out)
				}

				err = FlattenDir(tmpDir, output)

				return nil
			},
		},
	}

	err := app.Run(os.Args)
	utils.Assert(err)
}
