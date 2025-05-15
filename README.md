# asset-watcher

A command-line utility to fetch a list of IP addresses across a Google Cloud organization through Google Asset API.

[![license](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/andreygrechin/asset-watcher/blob/main/LICENSE)

## Features

- Collect `compute.googleapis.com/Address` assets.
- Filter by projects and a status.
- Output in a JSON or table format.

## Installation

### go install

```shell
go install github.com/andreygrechin/asset-watcher@latest
```

### Homebrew tap

You may also install the latest version of `asset-watcher` using the Homebrew tap:

```shell
brew install andreygrechin/tap/asset-watcher

# to update, run
brew update
brew upgrade asset-watcher
```

### Manually

Download the pre-compiled binaries from [the releases page](https://github.com/andreygrechin/asset-watcher/releases/) and copy them to a desired location.

## Usage

```shell
gcloud auth login && gcloud auth application-default login
export ASSET_WATCHER_ORG_ID=012345678912345
export ASSET_WATCHER_DEBUG=[true|false]
export ASSET_WATCHER_OUTPUT_FORMAT=[table|json]
export ASSET_WATCHER_EXCLUDE_RESERVED=[true|false]
export ASSET_WATCHER_EXCLUDE_PROJECTS=project-id-1,project-id-2
export ASSET_WATCHER_INCLUDE_PROJECTS=project-id-3,project-id-4
./asset-watcher
```

### Run in a local Docker container

```shell
make docker
docker run -it --rm \
  -e ASSET_WATCHER_ORG_ID=012345678912345 \
  -e ASSET_WATCHER_DEBUG=true \
  -e ASSET_WATCHER_OUTPUT_FORMAT=table \
  -e ASSET_WATCHER_EXCLUDE_RESERVED=true \
  -e ASSET_WATCHER_EXCLUDE_PROJECTS=project-id-1,project-id-2 \
  -e ASSET_WATCHER_INCLUDE_PROJECTS=project-id-3,project-id-4 \
  -v $HOME/.config/gcloud/application_default_credentials.json:/root/.config/gcloud/application_default_credentials.json \
  asset-watcher:latest
```

## License

This project is licensed under MIT licenses —  [MIT License](LICENSE).

`SPDX-License-Identifier: MIT`
