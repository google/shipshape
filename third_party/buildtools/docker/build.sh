#!/bin/sh -e

cd "$(dirname "$0")"
./get-deps.sh
docker build -t google/kythe-campfire .
