#!/bin/bash

BUILD_DIR="bin"
MAIN_DIR_PREFIX="cmd"

if [ ! -d $BUILD_DIR ];then
    mkdir -p $BUILD_DIR
fi

go build -o $BUILD_DIR/keygen $MAIN_DIR_PREFIX/keygen/main.go
go build -o $BUILD_DIR/client $MAIN_DIR_PREFIX/client/*.go
go build -o $BUILD_DIR/anti996 $MAIN_DIR_PREFIX/anti996/*.go
go build -o $BUILD_DIR/dbbrowser $MAIN_DIR_PREFIX/dbbrowser/main.go

