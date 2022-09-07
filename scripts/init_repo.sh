#!/bin/bash
set -uex

git submodule update --init --recursive
# if [ -z ${CI+x} ]; then git pull --recurse-submodules; fi
