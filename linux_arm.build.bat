set GOOS=linux
set GOARCH=arm64
set GOARM=7
go build -o bamfa-remote -ldflags="-s -w" main.go
pause