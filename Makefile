# Default settings for Linux
LUA_INCLUDE = ./lua-5.1.5/src
LIBS = -L./lua-5.1.5/src/ -llua -L. -lluaWebserver -Wl,-rpath,'$$ORIGIN'
CC = gcc
CFLAGS = -shared -fPIC -O2
TARGET = goLuaWebserver.so
SOURCE = goLuaWebserver.c
GO_BUILD_MODE = -buildmode=c-shared

# Platform-specific adjustments
ifeq ($(OS),Windows_NT)
    CC = x86_64-w64-mingw32-gcc
    TARGET = goLuaWebserver.dll
    GO_BUILD_MODE = -buildmode=c-shared -ldflags "-extldflags '-static'"
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Darwin)
        CC = clang
        CFLAGS = -shared -undefined dynamic_lookup -fPIC -O2
        TARGET = goLuaWebserver.dylib
    endif
    ifeq ($(UNAME_S),Linux)
        # Linux specific settings are already set as defaults
    endif
endif

# Target-specific flags for ARM architectures could be added here

# Default target
all: libluaWebserver.so $(TARGET)

libluaWebserver.so: luaWebserver.go
	go build -o libluaWebserver.so $(GO_BUILD_MODE) luaWebserver.go

$(TARGET): $(SOURCE) libluaWebserver.so
	$(CC) $(CFLAGS) -o $(TARGET) $(SOURCE) -I$(LUA_INCLUDE) $(LIBS)

clean:
	rm -f $(TARGET) libluaWebserver.so libLuaWebserver.h

