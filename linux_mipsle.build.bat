set GOOS=linux
set GOARCH=mipsle
set GOMIPS=softfloat
go build -o bamfa-remote -ldflags="-s -w" main.go
pause