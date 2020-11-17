# ghsync

[![CircleCI](https://circleci.com/gh/izumin5210/ghsync/tree/master.svg?style=svg)](https://circleci.com/gh/izumin5210/ghsync/tree/master)
[![latest](https://img.shields.io/github/release/izumin5210/ghsync.svg)](https://github.com/izumin5210/ghsync/releases/latest)
[![license](https://img.shields.io/github/license/izumin5210/ghsync.svg)](./LICENSE)


## Installation

- macOS (for Homebrew users)
    - `brew install izumin5210/tools/ghsync`
- macOS, Linux
    - ```bash
      cd /tmp
      curl -Lo grapi https://github.com/izumin5210/ghsync/releases/download/v0.0.8/ghsync_linux_amd64.tar.gz
      tar -xf grapi
      sudo mv ghsync_linux_amd64/ghsync /usr/local/bin
      rm grapi
      rm -r ghsync_linux_amd64
      ```
- others
    - `go get github.com/izumin5210/ghsync/cmd/ghsync`
