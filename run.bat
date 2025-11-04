@echo off
REM Run Ferret compiler directly with go run

if "%~1"=="" (
    echo Usage: run.bat ^<file.fer^>
    echo Example: run.bat test/simple.fer
    exit /b 1
)

go run main.go %*
