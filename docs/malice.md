# Use with [MaliceIO](https://github.com/maliceio/malice)

## Tell MaliceIO to watch the current directory

```bash
$ malice watch .
```

## Use `get-mauled` to download to the current directory

```bash
$ docker run --rm -it -v `pwd`:/malware malice/get-mauled the-zoo
```

## Watch the fun begin :sunglasses: in another terminal

```bash
$ watch -n1 docker ps -a
```

## You can also watch the data stream into [elasticsearch](https://github.com/maliceio/elasticsearch) by checking out the [kibana](https://github.com/maliceio/kibana) UI

```bash
$ malice elk
```
