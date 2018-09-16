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

	humanize "github.com/dustin/go-humanize"
	"github.com/malice-plugins/pkgs/utils"
	"github.com/minio/minio-go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	contagioDumpURL   = "https://www.dropbox.com/sh/i6ed6v32x0fp94z/AAAQvOsOvbWrOs8T3_ZTXqQya?dl=1"
	theZooURL         = "https://github.com/ytisf/theZoo/archive/master.zip"
	malwareSamplesURL = "https://github.com/fabrimagic72/malware-samples/archive/master.zip"
)

var (
	// Version stores the plugin's version
	Version string

	// BuildTime stores the plugin's build time
	BuildTime string

	// local storage
	outputDir string
	// cloud storage
	storageURL    string
	storageTLS    bool
	storageZone   string
	storageBucket string
	storageID     string
	storageKey    string
	mc            *minio.Client
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
		return errors.Wrapf(err, "failed to read dir %s", src)
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

// PutDir puts all files into cloud storage
func PutDir(ctx context.Context, srcDir string) error {

	var count uint64
	var totalSize int64

	err := filepath.Walk(srcDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "failed to get info on file: %s", path)
			}
			if !info.IsDir() {
				n, err := mc.FPutObjectWithContext(ctx, storageBucket, filepath.Base(path), path, minio.PutObjectOptions{})
				if err != nil {
					return errors.Wrapf(err, "failed to write %s to cloud storage", filepath.Base(path))
				}
				count++
				totalSize = totalSize + n
				log.WithField("bytes", n).Debugf("wrote %s to cloud storage", filepath.Base(path))
			}
			return nil
		})
	if err != nil {
		return errors.Wrapf(err, "failed to walk directory: %s", srcDir)
	}
	log.WithFields(log.Fields{
		"count":      count,
		"total_size": humanize.Bytes(uint64(totalSize)),
	}).Info("malware successfully sent to cloud storage")

	return nil
}

func downloadFromURL(url string, tmpfile *os.File) error {
	// Download file
	log.WithField("url", url).Info("downloading file")
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

func findAllZips(dir string) ([]string, error) {
	var zips []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
			return errors.Wrapf(err, "failure accessing a path %q", dir)
		}
		log.Debugf("visited file: %q", path)
		if !info.IsDir() && (filepath.Ext(path) == ".zip" || filepath.Ext(path) == ".7z") {
			zips = append(zips, path)
		}
		return nil
	})
	if err != nil {
		return []string{}, errors.Wrapf(err, "failed to walk dir %s", dir)
	}

	return zips, nil
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

func downloadAndUnzip(ctx context.Context, url, password string) error {

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

	if mc != nil {
		return PutDir(ctx, tmpDir)
	}
	return FlattenDir(tmpDir, outputDir)
}

func exists(file string) bool {
	for _, p := range strings.Split(os.Getenv("PATH"), ":") {
		_, err := os.Stat(filepath.Join(p, file))
		if err == nil {
			return true
		}
	}
	return false
}

// SetUpDestination configures the malware destination
func SetUpDestination() error {
	var err error

	if len(storageURL) > 0 {
		mc, err = minio.New(storageURL, storageID, storageKey, storageTLS)
		if err != nil {
			return errors.Wrap(err, "unable to create s3/minio client")
		}

		err = mc.MakeBucket(storageBucket, storageZone)
		if err != nil {
			// Check to see if we already own this bucket (which happens if you run this twice)
			exists, err := mc.BucketExists(storageBucket)
			if err == nil && exists {
				log.Debugf("bucket %s already exists", storageBucket)
			} else {
				return errors.Wrapf(err, "unable to create bucket: %s", storageBucket)
			}
		} else {
			log.Infof("successfully created bucket %s", storageBucket)
		}
	} else if len(outputDir) > 0 {
		if _, err = os.Stat(outputDir); os.IsNotExist(err) {
			return errors.Wrapf(err, "directory %s doesn't exist", outputDir)
		}
	} else {
		outputDir, err = os.Getwd()
		if err != nil {
			return errors.Wrap(err, "unable to get current working directory")
		}
	}
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
			Value:  60 * 5,
			Usage:  "malice plugin timeout (in seconds)",
			EnvVar: "MALICE_TIMEOUT",
		},
		cli.StringFlag{
			Name:        "output, o",
			Usage:       "set output directory",
			EnvVar:      "MALICE_OUTPUT_DIRECTORY",
			Destination: &outputDir,
		},
		cli.StringFlag{
			Name:        "store-url",
			Usage:       "s3 or minio file server url",
			EnvVar:      "MALICE_STORAGE_URL",
			Destination: &storageURL,
		},
		cli.BoolFlag{
			Name:        "store-tls",
			Usage:       "enable secure (HTTPS) access",
			EnvVar:      "MALICE_STORAGE_TLS",
			Destination: &storageTLS,
		},
		cli.StringFlag{
			Name:        "store-zone",
			Value:       "us-east-1",
			Usage:       "s3 or minio availbility zone location",
			EnvVar:      "MALICE_STORAGE_ZONE",
			Destination: &storageZone,
		},
		cli.StringFlag{
			Name:        "store-bucket",
			Value:       "malice",
			Usage:       "name of the minio or s3 bucket",
			EnvVar:      "MALICE_STORAGE_BUCKET",
			Destination: &storageBucket,
		},
		cli.StringFlag{
			Name:        "store-id",
			Usage:       "user ID that uniquely identifies your account",
			EnvVar:      "MALICE_STORAGE_ID",
			Destination: &storageID,
		},
		cli.StringFlag{
			Name:        "store-key",
			Usage:       "secret key is the password to your account",
			EnvVar:      "MALICE_STORAGE_KEY",
			Destination: &storageKey,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Gotta' Catch Em' All",
			Action: func(c *cli.Context) error {

				// var err error

				// if c.GlobalBool("verbose") {
				// 	log.SetLevel(log.DebugLevel)
				// }

				// if !exists("7z") {
				// 	return fmt.Errorf("you need to install 7zip to use get-mauled")
				// }

				// ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.GlobalInt("timeout"))*time.Second)
				// defer cancel()

				// err = SetUpDestination()
				// if err != nil {
				// 	return errors.Wrap(err, "failed to setup malware destination")
				// }

				log.Error("this command hasn't been implimented yet")

				return nil
			},
		},
		{
			Name:    "the-zoo",
			Aliases: []string{"z"},
			Usage:   "Download and Unzip The Zoo Malware",
			Action: func(c *cli.Context) error {

				var err error

				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}

				if !exists("7z") {
					return fmt.Errorf("you need to install 7zip to use get-mauled")
				}

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.GlobalInt("timeout"))*time.Second)
				defer cancel()

				err = SetUpDestination()
				if err != nil {
					return errors.Wrap(err, "failed to setup malware destination")
				}

				tmpfile, err := ioutil.TempFile("", "thezoo")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp file")
				}

				err = downloadFromURL(theZooURL, tmpfile)
				if err != nil {
					return errors.Wrapf(err, "downloading %s failed", theZooURL)
				}

				if err := tmpfile.Close(); err != nil {
					return errors.Wrap(err, "failed to close tmp file")
				}

				zipDir, err := ioutil.TempDir("", "thezoo_zip")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp directory")
				}
				defer os.RemoveAll(zipDir)

				out, err := unzip(ctx, tmpfile.Name(), "", zipDir)
				if err != nil {
					return errors.Wrapf(err, "unzipping %s failed", tmpfile.Name())
				}
				log.Debug(out)
				os.Remove(tmpfile.Name())

				log.Debugf("looking for zips in %s", zipDir)
				zipFiles, err := findAllZips(filepath.Join(zipDir, "theZoo-master/malwares/Binaries"))
				if err != nil {
					return errors.Wrapf(err, "failed to find all zips in directory: %s", zipDir)
				}

				tmpDir, err := ioutil.TempDir("", "getmauled")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp directory")
				}
				defer os.RemoveAll(tmpDir)

				for _, zipFile := range zipFiles {
					fmt.Println(zipFile)
					out, _ := unzip(ctx, zipFile, "infected", tmpDir)
					if err != nil {
						return errors.Wrapf(err, "unzipping %s failed", zipFile)
					}
					log.Debug(out)
				}

				if mc != nil {
					return PutDir(ctx, tmpDir)
				}
				return FlattenDir(tmpDir, outputDir)
			},
		},
		{
			Name:    "contagio",
			Aliases: []string{"c"},
			Usage:   "Download and Unzip contagiodump Malware",
			Action: func(c *cli.Context) error {

				var err error

				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}

				if !exists("7z") {
					return fmt.Errorf("you need to install 7zip to use get-mauled")
				}

				// increase timeout because it's downloading ~3GBs
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.GlobalInt("timeout"))*30*time.Second)
				defer cancel()

				err = SetUpDestination()
				if err != nil {
					return errors.Wrap(err, "failed to setup malware destination")
				}

				tmpfile, err := ioutil.TempFile("", "contagio")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp file")
				}

				err = downloadFromURL(contagioDumpURL, tmpfile)
				if err != nil {
					return errors.Wrapf(err, "downloading %s failed", contagioDumpURL)
				}

				if err := tmpfile.Close(); err != nil {
					return errors.Wrap(err, "failed to close tmp file")
				}

				zipDir, err := ioutil.TempDir("", "contagio_zip")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp directory")
				}
				defer os.RemoveAll(zipDir)

				out, err := unzip(ctx, tmpfile.Name(), "", zipDir)
				if err != nil {
					return errors.Wrapf(err, "unzipping %s failed", tmpfile.Name())
				}
				log.Debug(out)
				os.Remove(tmpfile.Name())

				log.Debugf("looking for zips in %s", zipDir)
				zipFiles, err := findAllZips(zipDir)
				if err != nil {
					return errors.Wrapf(err, "failed to find all zips in directory: %s", zipDir)
				}

				tmpDir, err := ioutil.TempDir("", "getmauled")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp directory")
				}
				defer os.RemoveAll(tmpDir)

				for _, zipFile := range zipFiles {
					fmt.Println(zipFile)
					out, _ := unzip(ctx, zipFile, getPassword(zipFile), tmpDir)
					if err != nil {
						return errors.Wrapf(err, "unzipping %s failed", zipFile)
					}
					log.Debug(out)
				}

				if mc != nil {
					return PutDir(ctx, tmpDir)
				}
				return FlattenDir(tmpDir, outputDir)
			},
		},
		{
			Name:    "malware-samples",
			Aliases: []string{"m"},
			Usage:   "Download and Unzip Malware Samples",
			Action: func(c *cli.Context) error {

				var err error

				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}

				if !exists("7z") {
					return fmt.Errorf("you need to install 7zip to use get-mauled")
				}

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.GlobalInt("timeout"))*time.Second)
				defer cancel()

				err = SetUpDestination()
				if err != nil {
					return errors.Wrap(err, "failed to setup malware destination")
				}

				tmpfile, err := ioutil.TempFile("", "malware_samples")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp file")
				}

				err = downloadFromURL(malwareSamplesURL, tmpfile)
				if err != nil {
					return errors.Wrapf(err, "downloading %s failed", malwareSamplesURL)
				}

				if err := tmpfile.Close(); err != nil {
					return errors.Wrap(err, "failed to close tmp file")
				}

				zipDir, err := ioutil.TempDir("", "malware_samples_zip")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp directory")
				}
				defer os.RemoveAll(zipDir)

				out, err := unzip(ctx, tmpfile.Name(), "", zipDir)
				if err != nil {
					return errors.Wrapf(err, "unzipping %s failed", tmpfile.Name())
				}
				log.Debug(out)
				os.Remove(tmpfile.Name())

				log.Debugf("looking for zips in %s", zipDir)
				zipFiles, err := findAllZips(zipDir)
				if err != nil {
					return errors.Wrapf(err, "failed to find all zips in directory: %s", zipDir)
				}

				tmpDir, err := ioutil.TempDir("", "getmauled")
				if err != nil {
					return errors.Wrap(err, "failed to create tmp directory")
				}
				defer os.RemoveAll(tmpDir)

				for _, zipFile := range zipFiles {
					fmt.Println(zipFile)
					out, _ := unzip(ctx, zipFile, "infected", tmpDir)
					if err != nil {
						return errors.Wrapf(err, "unzipping %s failed", zipFile)
					}
					log.Debug(out)
				}

				if mc != nil {
					return PutDir(ctx, tmpDir)
				}
				return FlattenDir(tmpDir, outputDir)
			},
		},
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
			},
			Action: func(c *cli.Context) error {

				var err error

				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}

				if !exists("7z") {
					return fmt.Errorf("you need to install 7zip to use get-mauled")
				}

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.GlobalInt("timeout"))*time.Second)
				defer cancel()

				err = SetUpDestination()
				if err != nil {
					return errors.Wrap(err, "failed to setup malware destination")
				}

				if c.Args().Present() {
					err = downloadAndUnzip(ctx, c.Args().First(), c.String("password"))
					if err != nil {
						log.Fatal(err)
					}

				} else {
					log.Fatal(fmt.Errorf("please supply a URL to download and unzip"))
				}
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	utils.Assert(err)
}
