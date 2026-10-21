<#
.SYNOPSIS
Local development script for running the URL Shortener API and tests.
#>

param(
    [Parameter(Position=0)]
    [ValidateSet("run", "test", "build", "fmt", "vet")]
    [string]$Command = "run"
)

$env:Path = "C:\Users\SRINI\AppData\Local\Programs\go\bin;" + $env:Path

switch ($Command) {
    "run" {
        Write-Host "Starting URL Shortener API in local development mode..." -ForegroundColor Cyan
        go run ./cmd/api
    }
    "test" {
        Write-Host "Running automated tests..." -ForegroundColor Green
        go test -v ./...
    }
    "build" {
        Write-Host "Compiling production binary..." -ForegroundColor Green
        go build -ldflags="-s -w" -o bin/api.exe ./cmd/api
        Write-Host "Binary built at bin/api.exe" -ForegroundColor Green
    }
    "fmt" {
        Write-Host "Formatting code with gofmt..." -ForegroundColor Yellow
        gofmt -w .
    }
    "vet" {
        Write-Host "Running static analysis with go vet..." -ForegroundColor Yellow
        go vet ./...
    }
}
