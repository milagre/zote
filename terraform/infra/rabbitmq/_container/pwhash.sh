#!/usr/bin/env bash
SALT_HEX=$(echo $1 | base64 -d - | xxd -p)
PASS_HEX=$(echo -n $2 | xxd -p)
SHA=$(echo -n "$SALT_HEX $PASS_HEX" | xxd -r -p | sha512sum | head -c 128)
HASH=$(echo -n "$SALT_HEX $SHA" | xxd -r -p | base64 -w0)
echo "{\"hash\":\"${HASH}\"}"
