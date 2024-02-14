#!/bin/bash

# Set cross-compilation environment variables
export CC=i686-w64-mingw32-gcc
export AR=i686-w64-mingw32-ar rcu
export RANLIB=i686-w64-mingw32-ranlib
export LD=i686-w64-mingw32-ld
export LUA_T=lua.exe

# Call make with specified arguments
make -C "$(dirname "$0")" mingw "MYCFLAGS=-DLUA_BUILD_AS_DLL" "CC=i686-w64-mingw32-gcc"
