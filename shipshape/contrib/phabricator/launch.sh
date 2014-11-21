#!/bin/bash -e
export LOG=file
# wrap docker spawns bash so exit it
echo exit | wrapdocker
node /shipshape/server.js
