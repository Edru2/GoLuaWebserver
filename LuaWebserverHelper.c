#include "LuaWebserverHelper.h"
#include <lauxlib.h>
#include <lstate.h>
#include <lua.h>
#include <lualib.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>


LuaHttpResponse* callLuaFunc(lua_State* L, int luaRef, HttpRequest* request)
{
    if (L == NULL || request == NULL) {
        return NULL;
    }
    LuaHttpResponse* response = (LuaHttpResponse*)malloc(sizeof(LuaHttpResponse));
    if (!response) {
        return NULL;
    }

    lua_settop(L, 0);
    lua_rawgeti(L, LUA_REGISTRYINDEX, luaRef);
    lua_newtable(L);
    lua_pushstring(L, request->method);
    lua_setfield(L, -2, "method");
    lua_pushstring(L, request->path);
    lua_setfield(L, -2, "path");
    lua_pushstring(L, request->url);
    lua_setfield(L, -2, "url");
    lua_pushinteger(L, request->contentLength);
    lua_setfield(L, -2, "contentLength");
    lua_pushstring(L, request->host);
    lua_setfield(L, -2, "host");
    lua_pushstring(L, request->remoteAddr);
    lua_setfield(L, -2, "remoteAddr");
    lua_pushstring(L, request->body);
    lua_setfield(L, -2, "body");
    lua_newtable(L);
    for (int i = 0; i < request->headersCount; ++i) {
        lua_pushstring(L, request->headersValues[i]);
        lua_setfield(L, -2, request->headersKeys[i]);
    }
    lua_setfield(L, -2, "Headers");
    lua_pushinteger(L, request->headersCount);
    lua_setfield(L, -2, "headersCount");


    int error = 0;
    error = lua_pcall(L, 1, LUA_MULTRET, 0);
    if (error != 0) {
        const char* errorMsg = lua_tostring(L, -1);
        lua_pop(L, 1);  // Remove the error message from the stack
        free(response); // Avoid memory leak
        lua_pushfstring(L, "Error when calling hook function: %s", errorMsg);
        return NULL; // Indicate failure
    }

    if (lua_gettop(L) < 3) {
        free(response);
        lua_pushfstring(L, "Not enough arguments in function %s. Expected are %d got %d.", "http hook return", 3, lua_gettop(L));
        return NULL;
    }

    if (!lua_isnumber(L, 1)) {
        free(response);
        lua_pushfstring(L, "Argument %d (%s) must be a %s in function %s", 1, "status code", "integer", "http hook return");
        return NULL; // Indicate failure
    }
    if (!lua_isstring(L, 2)) {
        free(response);
        lua_pushfstring(L, "Argument %d (%s) must be a %s in function %s", 2, "response body", "string", "http hook return");
        return NULL; // Indicate failure
    }
    if (!lua_istable(L, 3)) {
        free(response);
        lua_pushfstring(L, "Argument %d (%s) must be a %s in function %s", 3, "header table", "table", "http hook return");
        return NULL; // Indicate failure
    }

    int statusCode = (int)lua_tonumber(L, 1);
    const char* responseBody = lua_tostring(L, -2);
    response->responseBody = malloc(strlen(responseBody) + 1);

    if (response->responseBody != NULL) {
        strcpy(response->responseBody, responseBody);
    }

    if (lua_istable(L, -1)) {
        lua_pushnil(L);
        int i = 0;
        while (lua_next(L, -2) != 0) {
            const char* headerName = lua_tostring(L, -2);
            const char* headerValue = lua_tostring(L, -1);

            if (i < 10) {
                strncpy(response->headersKeys[i], headerName, 255);
                response->headersKeys[i][255] = '\0';
                strncpy(response->headersValues[i], headerValue, 255);
                response->headersValues[i][255] = '\0';
            }

            lua_pop(L, 1);
            i++;
        }
        response->headersCount = i;
    }
    response->statusCode = statusCode;

    lua_pop(L, 3);
    return response;
}

void callLuaWebSocketFunc(lua_State* L, int luaRef, char* client, int messagetype, char* message)
{
    if (L == NULL || message == NULL) {
        return;
    }

    lua_settop(L, 0);
    lua_rawgeti(L, LUA_REGISTRYINDEX, luaRef);
    lua_pushstring(L, client);
    lua_pushinteger(L, messagetype);
    lua_pushstring(L, message);
    free(message);
    free(client);
    int error = 0;
    error = lua_pcall(L, 3, 0, 0);
    if (error != 0) {
        const char* errorMsg = lua_tostring(L, -1);
        lua_pop(L, 1);
        printf("%s\n", errorMsg);
        return;
    }
}
