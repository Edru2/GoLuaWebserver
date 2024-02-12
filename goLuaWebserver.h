// libluaWebsockets.h
#ifndef LIBLUAWEBSERVER_H
#define LIBLUAWEBSERVER_H

#include <lua.h>

// Define the LuaHttpResponse struct
typedef struct
{
    int statusCode;
    char* responseBody;
    char headersKeys[10][256];
    char headersValues[10][256];
    int headersCount;
} LuaHttpResponse;

// Function prototypes
LuaHttpResponse* callLuaFunc(lua_State* L, int luaRef, const char* method, const char* path);
void callLuaWebSocketFunc(lua_State* L, int luaRef, char* client, int messagetype, char* message);

#endif // LIBLUAWEBSERVER_H
