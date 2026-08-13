set GOOS=linux
set GOARCH=amd64
go build -o bin/reperlint ./cmd/reperlint/main.go
set GOOS=windows
set GOARCH=amd64
go build -o bin/reperlint.exe ./cmd/reperlint/main.go