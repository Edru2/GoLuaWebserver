// LuaWebserverHelper.h
#ifndef LUAWEBSERVERHELPER_H
#define LUAWEBSERVERHELPER_H

#include <lua.h>
#include <stdbool.h>

typedef struct
{
    int statusCode;
    char* responseBody;
    char headersKeys[20][256];
    char headersValues[20][256];
    int headersCount;
} LuaHttpResponse;

typedef struct
{
    char* method;
    char* path;
    char* url;
    char* proto;
    long contentLength;
    char* host;
    char* remoteAddr;
    char headersKeys[50][256];
    char headersValues[50][256];
    int headersCount;
    char* body;
} HttpRequest;

typedef struct
{
    char* msg;
    bool success;
    int id;
} Message;

typedef struct 
{
    Message errHandling;
    int clientCount;
    char** clientIds;
    char** paths;
} ClientInfo;

// Function prototypes
LuaHttpResponse* callLuaFunc(lua_State* L, int luaRef, HttpRequest* request);
void callLuaWebSocketFunc(lua_State* L, int luaRef, char* client, int messagetype, char* message);

#endif // LUAWEBSERVERHELPER_H
