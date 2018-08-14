![get-mauled logo](https://github.com/malice-plugins/get-mauled/blob/master/docs/maul-logo.png)

# get-mauled

[![Circle CI](https://circleci.com/gh/malice-plugins/get-mauled.png?style=shield)](https://circleci.com/gh/malice-plugins/get-mauled) [![License](http://img.shields.io/:license-mit-blue.svg)](http://doge.mit-license.org) [![Docker Stars](https://img.shields.io/docker/stars/malice/get-mauled.svg)](https://hub.docker.com/r/malice/get-mauled/) [![Docker Pulls](https://img.shields.io/docker/pulls/malice/get-mauled.svg)](https://hub.docker.com/r/malice/get-mauled/) [![Docker Image](https://img.shields.io/badge/docker%20image-27.3MB-blue.svg)](https://hub.docker.com/r/malice/get-mauled/)

> Download a Bunch of Malware for Demos and Testing

---

### Dependencies

- [malice/alpine](https://hub.docker.com/r/malice/alpine/)

## Installation

1.  Install [Docker](https://www.docker.io/).
2.  Download [trusted build](https://hub.docker.com/r/malice/get-mauled/) from public [DockerHub](https://hub.docker.com): `docker pull malice/get-mauled`

## Usage

```bash
$ docker run --rm -v /path/to/malware:/malware malice/get-mauled --help

Usage: get-mauled [OPTIONS] COMMAND [arg...]

Malice Demo Malware Downloader Plugin

Version: v0.1.0, BuildTime: 20180814

Author:
  blacktop - <https://github.com/blacktop>

Options:
  --verbose, -V  verbose output
  --help, -h     show help
  --version, -v  print the version

Commands:
  download  Download Malware
  help      Shows a list of commands or help for one command

Run 'get-mauled COMMAND --help' for more information on a command.
```

### Download Ze Malwarez :smiling_imp:

```bash
$ docker run --rm -it -v /path/to/malware:/malware malice/get-mauled download

INFO[0000] Downloading file: https://github.com/ytisf/theZoo/raw/master/malwares/Binaries/Duqu2/Duqu2.zip
```

```bash
ls -lah /path/to/malware
```

```bash
-rw-r--r--   1 blacktop  staff    13K Aug 13 22:49 2796a119171328e91648a73d95eb297edc220e8768f4bbba5fb7237122a988fc
-rw-r--r--   1 blacktop  staff    16K Aug 13 22:49 2a9a5afc342cde12c6eb9a91ad29f7afdfd8f0fb17b983dcfddceccfbc17af69
-rw-r--r--   1 blacktop  staff    17K Aug 13 22:49 2c9c3ddd4d93e687eb095444cef7668b21636b364bff55de953bdd1df40071da
-rw-r--r--   1 blacktop  staff    17K Aug 13 22:49 2ecb26021d21fcef3d8bba63de0c888499110a2b78e4caa6fa07a2b27d87f71b
-rw-r--r--   1 blacktop  staff    12K Aug 13 22:49 3536df7379660d931256b3cf49be810c0d931c3957c464d75e4cba78ba3b92e3
-rw-r--r--   1 blacktop  staff   254K Aug 13 22:49 52fe506928b0262f10de31e783af8540b6a0b232b15749d647847488acd0e17a
-rw-r--r--   1 blacktop  staff    18K Aug 13 22:49 5559fcc93eef38a1c22db66a3e0f9e9f026c99e741cc8b1a4980d166f2696188
-rw-r--r--   1 blacktop  staff    17K Aug 13 22:49 5ba187106567e8d036edd5ddb6763f89774c158d2a571e15d76572d8604c22a0
-rw-r--r--   1 blacktop  staff    25K Aug 13 22:49 6217cebf11a76c888cc6ae94f54597a877462ed70da49a88589a9197173cc072
-rw-r--r--   1 blacktop  staff    14K Aug 13 22:49 6b146e3a59025d7085127b552494e8aaf76450a19c249bfed0b4c09f328e564f
-rw-r--r--   1 blacktop  staff    17K Aug 13 22:49 6c803aac51038ce308ee085f2cd82a055aaa9ba24d08a19efb2c0fcfde936c34
-rw-r--r--   1 blacktop  staff    24K Aug 13 22:49 6de1bb58ae3c37876c6372208366f5548fcc647ffd19ad1d31cebd9069b8a559
-rw-r--r--   1 blacktop  staff    17K Aug 13 22:49 6e09e1a4f56ea736ff21ad5e188845615b57e1a5168f4bdaebe7ddc634912de9
-rw-r--r--   1 blacktop  staff   274K Aug 13 22:49 81cdbe905392155a1ba8b687a02e65d611b60aac938e470a76ef518e8cffd74d
-rw-r--r--   1 blacktop  staff    18K Aug 13 22:49 8e97c371633d285cd8fc842f4582705052a9409149ee67d97de545030787a192
-rw-r--r--   1 blacktop  staff    24K Aug 13 22:49 9900c91f6d754f15f73729ce5a4333a718463e24aa7e6192c7527ec5c80dac42
-rw-r--r--   1 blacktop  staff    27K Aug 13 22:49 bc4ae56434b45818f57724f4cd19354a13e5964fd097d1933a30e2e31c9bdfa5.bin
-rw-r--r--   1 blacktop  staff    17K Aug 13 22:49 c16410c49dc40a371be22773f420b7dd3cfd4d8205cf39909ad9a6f26f55718e
-rw-r--r--   1 blacktop  staff    17K Aug 13 22:49 d12cd9490fd75e192ea053a05e869ed2f3f9748bf1563e6e496e7153fb4e6c98
-rw-r--r--   1 blacktop  staff    25K Aug 13 22:49 d5c57788cf12b020c4083eb228911260b744a2a67c88662c9bab8faebca98fa2
-rw-r--r--   1 blacktop  staff    15K Aug 13 22:49 d8a849654ab97debaf28ae5b749c3b1ff1812ea49978713853333db48c3972c3
-rw-r--r--   1 blacktop  staff    25K Aug 13 22:49 e83c6c36dbd143ee0fd36aff30fb43529a34129817dc2530f251121527cbf4b4
```

### Use with [MaliceIO](https://github.com/maliceio/malice)

Tell MaliceIO to watch the current directory

```bash
$ malice watch .
```

Use `get-mauled` to download to the current directory

```bash
$ docker run --rm -it -v `pwd`:/malware malice/get-mauled download
```

Watch the fun begin :sunglasses: in another terminal

```bash
$ watch -n1 docker ps -a
```

You can also watch the data stream into [elasticsearch](https://github.com/maliceio/elasticsearch) by checking out the [kibana](https://github.com/maliceio/kibana) UI

```bash
$ malice elk
```

## Issues

Find a bug? Want more features? Find something missing in the documentation? Let me know! Please don't hesitate to [file an issue](https://github.com/malice-plugins/get-mauled/issues/new)

## CHANGELOG

See [`CHANGELOG.md`](https://github.com/malice-plugins/get-mauled/blob/master/CHANGELOG.md)

## Contributing

[See all contributors on GitHub](https://github.com/malice-plugins/get-mauled/graphs/contributors).

Please update the [CHANGELOG.md](https://github.com/malice-plugins/get-mauled/blob/master/CHANGELOG)

## Credits

- https://github.com/ytisf/theZoo
- https://contagiodump.blogspot.com/
- https://github.com/fabrimagic72/malware-samples

## TODO

- [ ] parse all zips at the URLs above
- [ ] create golang binary releases for people who hate docker :whale:

## License

MIT Copyright (c) 2018 **blacktop**
