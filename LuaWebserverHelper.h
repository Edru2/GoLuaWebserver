// LuaWebserverHelper.h
#ifndef LUAWEBSERVERHELPER_H
#define LUAWEBSERVERHELPER_H

#include <lua.h>

typedef struct
{
    int statusCode;
    char* responseBody;
    char headersKeys[10][256];
    char headersValues[10][256];
    int headersCount;
} LuaHttpResponse;

typedef struct {
    char* method;
    char* path;
    char* url;
    char* proto;
    long contentLength;
    char* host;
    char* remoteAddr;
    char headersKeys[10][256];
    char headersValues[10][256];
    int headersCount;
    char* body;
} HttpRequest;

// Function prototypes
LuaHttpResponse* callLuaFunc(lua_State* L, int luaRef, HttpRequest* request);
void callLuaWebSocketFunc(lua_State* L, int luaRef, char* client, int messagetype, char* message);

#endif // LUAWEBSERVERHELPER_H
