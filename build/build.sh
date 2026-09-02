#!/bin/bash

PROJECT_PATH=$(pwd)
BUILD_PATH="${PROJECT_PATH}/build/"
BIN_PATH="${PROJECT_PATH}/build/_result_lin"
CONFIGS_PATH="${BIN_PATH}/configs"

echo "path: ${BUILD_PATH}"

# CREATE DIRS
mkdir -p "${BIN_PATH}"
mkdir -p "${CONFIGS_PATH}"

# CLEAR BUILDS
rm -f "${BIN_PATH}"/*
rm -f "${CONFIGS_PATH}"/*

# BUILD BINS
go build -o "${BIN_PATH}/server" "${PROJECT_PATH}/cmd/server/main.go"
if [ $? -ne 0 ]; then
    echo "[ERROR] Server build failed!"
    exit 1
fi

go build -o "${BIN_PATH}/client" "${PROJECT_PATH}/cmd/client/main.go"
if [ $? -ne 0 ]; then
    echo "[ERROR] Client build failed!"
    exit 1
fi

# COPY CONFIGS
cp "${PROJECT_PATH}/configs/server.example.yaml" "${CONFIGS_PATH}/server.yaml"
cp "${PROJECT_PATH}/configs/client.example.yaml" "${CONFIGS_PATH}/client.yaml"