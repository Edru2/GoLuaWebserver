#!/bin/bash

# Set cross-compilation environment variables
export CC=mingw32-gcc
export AR=mingw32-ar rcu
export RANLIB=mingw32-ranlib
export LD=mingw32-ld
export LUA_T=lua.exe

# Call make with specified arguments
make -C "$(dirname "$0")" mingw "MYCFLAGS=-DLUA_BUILD_AS_DLL" "CC=mingw32-gcc"
