@echo off

set PROJECT_PATH=%cd%
set BUILD_PATH=%PROJECT_PATH%\build\
set BIN_PATH=%cd%\build\_bin
set CONFIGS_PATH=%cd%\build\_configs

echo path: %BUILD_PATH%

:: CREATE DIRS
if not exist %BIN_PATH% (
    mkdir %BIN_PATH%
)
if not exist %CONFIGS_PATH% (
    mkdir %CONFIGS_PATH%
)

:: CLEAR BUILDS
del /Q %BIN_PATH%\*
del /Q %CONFIGS_PATH%\*

:: BUILD BINS
go build -o %BIN_PATH%\server.exe %PROJECT_PATH%\cmd\server\main.go
if errorlevel 1 (
    echo [ERROR] Server build failed!
    exit /b 1
)

go build -o %BIN_PATH%\client.exe %PROJECT_PATH%\cmd\client\main.go
if errorlevel 1 (
    echo [ERROR] Server build failed!
    exit /b 1
)

:: COPY CONFIGS
copy %PROJECT_PATH%\configs\server.example.yaml %CONFIGS_PATH%\server.yaml
copy %PROJECT_PATH%\configs\client.example.yaml %CONFIGS_PATH%\client.yaml