# redis-stream-exp

> Playing with redis streams :)

## Quickstart

**Pre-requisites**: Docker or Podman

```sh
# Run services
podman compose up --build
# Open the grafana dashboard at http://localhost:3000
# and login with admin / admin123

# Run the k6 load test
podman compose --profile k6 up

# Tear down services
podman compose down
```

## Dev environment setup

### Install pre-requisites

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

### Run the app 

Make sure redis is running on port 6379, then:

```sh
cd app
go run *.go
```

### Run a load test 

```sh
cd k6
./run_load_test.sh
```
