# pctl
CLI for interacting with [profiles](https://github.com/weaveworks/profiles)

## Commands

### Search
```
$ export CATALOG_URL="https://gist.githubusercontent.com/bigkevmcd/dd211661f9b01fa42eade2737f5dc059/raw/8edc1f353bad00a55da009d1834e8455b2e3312f/testing.yaml"
$ pctl search --catalog-url=$CATALOG_URL nginx
searching for profiles matching "nginx":
weaveworks-nginx: This installs nginx.
```

## Development

### Build
To build the binary run `make build`

### Testing
To run the tests run `make test
