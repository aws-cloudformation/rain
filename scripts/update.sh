#!/bin/bash
# Update everything except pkl-go (v0.8.1 has a bad hash..?)
go get -u ./... github.com/apple/pkl-go@v0.8.0
