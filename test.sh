export CGO_CFLAGS="-I./lua-5.1.5/src"
export CGO_LDFLAGS="-L./lua-5.1.5/src -llua -L.-lLuaWebserverApi"

go tool cgo -exportheader luaWebserver.h luaWebserver.go
