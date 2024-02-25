package main

/*
#cgo CFLAGS: -I./external/luajit/src
#cgo LDFLAGS: -L. -lluaWebserverHelper
#include <lua.h>
#include <lualib.h>
#include <lauxlib.h>
#include "LuaWebserverHelper.h"
#include <stdlib.h>
#include <string.h>
typedef const char cchar_t;
*/
import "C"
import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
}

type PathFunction struct {
	FunctionRef C.int
	LuaState    *C.lua_State
	Clients     map[string]*Client
}

type Server struct {
	Paths            map[string]*PathFunction
	WebSocketClients map[string]*Client
	http.Server
}

var (
	servers = make(map[int]*Server)
	nextID  = 0
	mutex   = &sync.RWMutex{}
)

func generateUniqueID() string {
	return uuid.New().String()
}

//export Serve
func Serve(L *C.lua_State, serverID C.int, path *C.cchar_t, luaFuncRef C.int) C.Message {
	goServerId := int(serverID)

	mutex.RLock()
	server, exists := servers[goServerId]
	mutex.RUnlock()
	if !exists {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("Server with ID %d not found", serverID)),
			success: false,
		}
	}

	goPath := C.GoString(path)
	if goPath == "" {
		goPath = "/"
	}
	mutex.RLock()
	_, exists = server.Paths[goPath]
	mutex.RUnlock()
	if exists {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("Path %s exists already!", goPath)),
			success: false,
		}
	}

	mutex.Lock()
	server.Paths[goPath] = &PathFunction{
		FunctionRef: luaFuncRef,
		LuaState:    L,
	}
	mutex.Unlock()

	mux := server.Handler.(*http.ServeMux)
	mux.HandleFunc(goPath,
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != goPath {
				http.NotFound(w, r)
				return
			}
			luaState := server.Paths[goPath].LuaState
			luaFuncRef := server.Paths[goPath].FunctionRef
			if luaState == nil {
				return
			}
			mutex.Lock()
			statusCode, responseBody, headers := callLuaFunction(luaState, luaFuncRef, r, goPath)
			mutex.Unlock()

			if statusCode == 404 && responseBody == "" && len(headers) == 0 {
				http.NotFound(w, r)
				return
			}
			for key, value := range headers {
				w.Header().Set(key, value)
			}
			w.WriteHeader(statusCode)
			w.Write([]byte(responseBody))
		})
	return C.Message{
		success: true,
	}

}

func callLuaFunction(L *C.lua_State, luaFuncRef C.int, r *http.Request, path string) (int, string, map[string]string) {
	bodyBytes, _ := io.ReadAll(r.Body)
	bodyContent := string(bodyBytes)
	cReq := C.HttpRequest{
		method:        C.CString(r.Method),
		path:          C.CString(path),
		url:           C.CString(r.URL.String()),
		proto:         C.CString(r.Proto),
		contentLength: C.long(r.ContentLength),
		host:          C.CString(r.Host),
		remoteAddr:    C.CString(r.RemoteAddr),
		headersCount:  C.int(len(r.Header)),
		body:          C.CString(bodyContent),
	}

	defer func() {
		C.free(unsafe.Pointer(cReq.method))
		C.free(unsafe.Pointer(cReq.path))
		C.free(unsafe.Pointer(cReq.url))
		C.free(unsafe.Pointer(cReq.proto))
		C.free(unsafe.Pointer(cReq.host))
		C.free(unsafe.Pointer(cReq.remoteAddr))
		C.free(unsafe.Pointer(cReq.body))
	}()
	maxHeaders := 50
	if len(r.Header) < maxHeaders {
		maxHeaders = len(r.Header)
	}
	cReq.headersCount = C.int(maxHeaders)

	i := 0
	for key, values := range r.Header {
		if i >= maxHeaders {
			break
		}
		keyCStr := C.CString(key)
		valueCStr := C.CString(strings.Join(values, "|"))
		defer C.free(unsafe.Pointer(keyCStr))
		defer C.free(unsafe.Pointer(valueCStr))

		C.strncpy(&cReq.headersKeys[i][0], keyCStr, 255)
		cReq.headersKeys[i][255] = 0 // Ensure null termination
		C.strncpy(&cReq.headersValues[i][0], valueCStr, 255)
		cReq.headersValues[i][255] = 0 // Ensure null termination
		i++
	}
	///////
	cResponse := C.callLuaFunc(L, luaFuncRef, &cReq)
	///////
	responseHeaders := make(map[string]string)

	if cResponse == nil {
		errMsg := C.GoString(C.lua_tolstring(L, -1, nil))
		return 500, fmt.Sprintf("Internal Server Error: %s", errMsg), responseHeaders
	}

	defer C.free(unsafe.Pointer(cResponse))
	statusCode := int(cResponse.statusCode)
	var responseBody string
	if cResponse.responseBody != nil {
		defer C.free(unsafe.Pointer(cResponse.responseBody))
		responseBody = C.GoString(cResponse.responseBody)
	}
	for i := 0; i < int(cResponse.headersCount); i++ {
		key := C.GoString(&cResponse.headersKeys[i][0])
		value := C.GoString(&cResponse.headersValues[i][0])
		responseHeaders[key] = value
	}

	return statusCode, responseBody, responseHeaders
}

// Check if the address is available by trying to listen on it
func isAddressAvailable(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	listener.Close()
	return nil
}

//export StartServer
func StartServer(address, certFile, keyFile *C.cchar_t) C.Message {
	serverAddress := C.GoString(address)
	if err := isAddressAvailable(serverAddress); err != nil {
		errMessage := fmt.Sprintf("Failed to create server at %s, reason %s", C.GoString(address), err.Error())
		log.Printf(errMessage)
		return C.Message{
			msg:     C.CString(errMessage),
			success: false,
		}
	}
	id := nextID
	nextID++
	mux := http.NewServeMux()
	server := &Server{
		Server: http.Server{
			Addr:           serverAddress,
			Handler:        mux,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Paths:            make(map[string]*PathFunction),
		WebSocketClients: make(map[string]*Client),
	}
	servers[id] = server
	go func() {
		defer delete(servers, id)
		if certFile == nil || keyFile == nil {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Println("ListenAndServe: ", err)
			}
		} else {
			if err := server.ListenAndServeTLS(C.GoString(certFile), C.GoString(keyFile)); err != nil && err != http.ErrServerClosed {
				log.Println("ListenAndServe: ", err)
			}
		}
	}()
	return C.Message{
		success: true,
		id:      C.int(id),
	}
}

//export ServeWebSocket
func ServeWebSocket(L *C.lua_State, serverID C.int, path *C.cchar_t, luaFuncRef C.int) C.Message {
	goServerId := int(serverID)

	server, exists := servers[goServerId]
	if !exists {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("Server with ID %d not found", serverID)),
			success: false,
		}
	}
	goPath := C.GoString(path)
	if goPath == "" {
		goPath = "/"
	}
	mutex.RLock()
	_, exists = server.Paths[goPath]
	mutex.RUnlock()

	if exists {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("Path %s exists already!", goPath)),
			success: false,
		}
	}
	mutex.Lock()
	server.Paths[goPath] = &PathFunction{
		FunctionRef: luaFuncRef,
		LuaState:    L,
		Clients:     make(map[string]*Client),
	}
	mutex.Unlock()

	mux := server.Handler.(*http.ServeMux)

	var upgrader = websocket.Upgrader{}
	mux.HandleFunc(goPath,
		func(w http.ResponseWriter, r *http.Request) {
			luaState := server.Paths[goPath].LuaState
			luaFuncRef := server.Paths[goPath].FunctionRef
			if luaState == nil {
				return
			}
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Print("upgrade:", err)
				return
			}
			defer c.Close()

			clientID := generateUniqueID()
			client := Client{ID: clientID, Conn: c}
			mutex.Lock()
			server.WebSocketClients[clientID] = &client
			server.Paths[goPath].Clients[clientID] = &client
			mutex.Unlock()
			defer delete(server.Paths[goPath].Clients, clientID)
			defer delete(server.WebSocketClients, clientID)

			for {
				mt, message, err := c.ReadMessage()
				if mt == websocket.CloseMessage {
					log.Printf("Client: %s disconnected.", clientID)
					break
				}
				if err != nil {
					log.Println("read:", err)
					break
				}
				mutex.Lock()
				C.callLuaWebSocketFunc(luaState, luaFuncRef, C.CString(clientID), C.int(mt), C.CString(string(message)))
				mutex.Unlock()
			}
		})
	return C.Message{
		success: true,
	}
}

//export WriteToWebSocketClient
func WriteToWebSocketClient(serverID C.int, clientID *C.cchar_t, message *C.cchar_t) C.Message {
	goClientId := C.GoString(clientID)
	goServerId := int(serverID)
	goMessage := C.GoString(message)

	server, exists := servers[goServerId]
	if !exists {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("Server with ID %d not found", serverID)),
			success: false,
		}
	}

	client, exists := server.WebSocketClients[goClientId]
	if !exists {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("Client with ID %s not found", goClientId)),
			success: false,
		}
	}

	err := client.Conn.WriteMessage(websocket.TextMessage, []byte(goMessage))
	if err != nil {
		return C.Message{
			msg:     C.CString(fmt.Sprint("Write to websocket error:", err)),
			success: false,
		}

	}
	return C.Message{
		success: true,
	}
}

//export BroadcastToWebSocket
func BroadcastToWebSocket(serverID C.int, path *C.cchar_t, message *C.cchar_t) C.Message {
	goServerId := int(serverID)
	goMessage := C.GoString(message)

	server, exists := servers[goServerId]
	if !exists {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("Server with ID %d not found", serverID)),
			success: false,
		}
	}
	goPath := C.GoString(path)
	if goPath == "" {
		goPath = "/"
	}
	mutex.RLock()
	_, exists = server.Paths[goPath]
	mutex.RUnlock()

	if !exists || len(server.Paths[goPath].Clients) == 0 {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("No path or no clients at '%s'", goPath)),
			success: false,
		}
	}

	var errMsg strings.Builder
	for _, client := range server.Paths[goPath].Clients {
		err := client.Conn.WriteMessage(websocket.TextMessage, []byte(goMessage))
		if err != nil {
			errMsg.WriteString(err.Error() + "\n")
		}
	}

	if errMsg.Len() > 0 {
		return C.Message{
			msg:     C.CString(fmt.Sprint("Broadcast to websocket error(s):", errMsg.String())),
			success: false,
		}
	}

	return C.Message{
		success: true,
	}
}

//export GetWebSocketClients
func GetWebSocketClients(serverID C.int) C.ClientInfo {
	goServerId := int(serverID)

	cClientInfo := C.ClientInfo{
		errHandling: C.Message{success: true},
	}

	server, exists := servers[goServerId]
	if !exists {
		cClientInfo.errHandling = C.Message{msg: C.CString(fmt.Sprintf("Server with ID %d not found", serverID)),
			success: false}

		return cClientInfo
	}
	type clientInfo struct {
		clientId string
		path     string
	}
	var clientList []clientInfo
	for path, pathFunction := range server.Paths {
		if len(pathFunction.Clients) != 0 {
			for clientId, _ := range pathFunction.Clients {
				clientList = append(clientList, clientInfo{clientId: clientId, path: path})
			}
		}
	}

	clientCount := len(clientList)
	cClientInfo.clientCount = C.int(clientCount)
	cClientInfo.clientIds = (**C.char)(C.calloc(C.size_t(clientCount), C.size_t(unsafe.Sizeof(uintptr(0)))))
	cClientInfo.paths = (**C.char)(C.calloc(C.size_t(clientCount), C.size_t(unsafe.Sizeof(uintptr(0)))))
	for i, client := range clientList {
		clientId := C.CString(client.clientId)
		clientPath := C.CString(client.path)

		offsetClientId := (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cClientInfo.clientIds)) + uintptr(i)*unsafe.Sizeof(clientId)))
		*offsetClientId = clientId

		offsetClientPath := (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cClientInfo.paths)) + uintptr(i)*unsafe.Sizeof(clientPath)))
		*offsetClientPath = clientPath
	}

	return cClientInfo
}

//export ServeFiles
func ServeFiles(serverID C.int, path *C.cchar_t, dir *C.cchar_t) C.Message {
	goServerId := int(serverID)
	server, exists := servers[goServerId]
	if !exists {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("Server with ID %d not found", serverID)),
			success: false,
		}
	}
	goDir := C.GoString(dir)
	goPath := C.GoString(path)
	if goPath == "" {
		goPath = "/"
	}
	mutex.RLock()
	_, exists = server.Paths[goPath]
	mutex.RUnlock()
	if exists {
		return C.Message{
			msg:     C.CString(fmt.Sprintf("Path %s exists already!", goPath)),
			success: false,
		}
	}

	mutex.Lock()
	server.Paths[goPath] = &PathFunction{}
	mutex.Unlock()

	fileServer := http.FileServer(http.Dir(goDir))
	mux := server.Handler.(*http.ServeMux)
	mux.Handle(goPath, http.StripPrefix(goPath, fileServer))
	return C.Message{
		success: true,
	}
}

//export StopServer
func StopServer(serverID C.int) C.Message {
	id := int(serverID)
	err := GoStopServer(id)
	if err != nil {
		return C.Message{
			msg:     C.CString(err.Error()),
			success: false,
		}

	}
	return C.Message{
		success: true,
	}

}

func GoStopServer(id int) error {
	server, exists := servers[id]
	if !exists {
		log.Printf("Server with ID %d not found", id)
		return fmt.Errorf("Server with ID %d not found", id)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for clientID, client := range server.WebSocketClients {
		err := client.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Printf("Failed to write close message to client %s: %v", clientID, err)
		}
		err = client.Conn.Close()
		if err != nil {
			log.Printf("Failed to close client %s: %v", clientID, err)
		}
	}
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown failed: %+v", err)
		return fmt.Errorf("Shutdown failed: %+v", err)
	}
	delete(servers, id)
	log.Printf("Server with ID %d shut down successfully", id)
	return nil
}

//export StopAllServers
func StopAllServers() {
	for id := range servers {
		GoStopServer(id)
	}
}

//export StopLuaStateFunctions
func StopLuaStateFunctions(L *C.lua_State) {
	for _, server := range servers {
		for _, pathFunction := range server.Paths {
			if L == pathFunction.LuaState {
				pathFunction.FunctionRef = 0
				pathFunction.LuaState = nil
			}
		}
	}
}

func main() {}
