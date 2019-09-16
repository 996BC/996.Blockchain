#!/bin/bash

find . -name "*_test.go" | xargs dirname | sort | uniq | xargs go test -count=1
