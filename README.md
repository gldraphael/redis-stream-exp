# redis-stream-exp

> Playing with redis streams :)

## Install pre-requisites

The following are required:

* The `Go` toolchain
* `redis` running & available on the default port 6379
* `k6`

Feel free to adapt the following script to setup your machine:

```sh
brew install go k6 nvm
nvm install 25
podman run -d --name redis -p 6379:6379 redis
```

## Run the app

```sh
cd app
go run *.go
```

## Run a load test

```sh
cd k6
./run_load_test.sh
```
