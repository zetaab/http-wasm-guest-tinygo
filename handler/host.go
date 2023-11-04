package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"unsafe"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/internal/imports"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/internal/mem"
)

// wasmHost implements api.Host with imported WebAssembly functions.
type wasmHost struct{}

// compile-time check to ensure wasmHost implements api.Host.
var _ api.Host = wasmHost{}

// EnableFeatures implements the same method as documented on api.Host.
func (wasmHost) EnableFeatures(features api.Features) api.Features {
	return imports.EnableFeatures(features)
}

// GetConfig implements the same method as documented on api.Host.
func (wasmHost) GetConfig() []byte {
	return mem.GetBytes(imports.GetConfig)
}

// LogEnabled implements the same method as documented on api.Host.
func (h wasmHost) LogEnabled(level api.LogLevel) bool {
	if enabled := imports.LogEnabled(level); enabled == 1 {
		return true
	}
	return false
}

// Log implements the same method as documented on api.Host.
func (wasmHost) Log(level api.LogLevel, message string) {
	if len(message) == 0 {
		return // don't incur host call overhead
	}
	ptr, size := mem.StringToPtr(message)
	imports.Log(level, ptr, size)
	runtime.KeepAlive(message) // keep message alive until ptr is no longer needed.
}

// HTTPRequest implements the same method as documented on api.Host.
func (wasmHost) HTTPRequest(method string, uri string, body string) (*http.Response, error) {
	methodPtr, methodSize := mem.StringToPtr(method)
	uriPtr, uriSize := mem.StringToPtr(uri)
	bodyPtr, bodySize := mem.StringToPtr(body)

	readBufLimit := uint32(2048)
	readBuf := make([]byte, readBufLimit)
	readBufPtr := uint32(uintptr(unsafe.Pointer(&readBuf[0])))

	readBodyBufLimit := uint32(2048)
	readBodyBuf := make([]byte, readBodyBufLimit)
	readBodyBufPtr := uint32(uintptr(unsafe.Pointer(&readBodyBuf[0])))
	size := imports.HTTPRequest(readBufPtr, readBufLimit, methodPtr, methodSize, uriPtr, uriSize, bodyPtr, bodySize, readBodyBufPtr, readBodyBufLimit)

	result := make([]byte, size)
	copy(result, readBuf)

	type httpResponse struct {
		Code    uint32
		Body    uint32
		Headers http.Header
	}

	data := &httpResponse{}
	err := json.Unmarshal(result, &data)
	if err != nil {
		return nil, err
	}
	responseBody := make([]byte, data.Body)
	copy(responseBody, readBodyBuf)
	if data.Code == 0 {
		return nil, fmt.Errorf("%s", responseBody)
	}
	response := &http.Response{
		Status:        fmt.Sprintf("%v %v", data.Code, http.StatusText(int(data.Code))),
		StatusCode:    int(data.Code),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          ioutil.NopCloser(bytes.NewBuffer(responseBody)),
		ContentLength: int64(data.Body),
		Header:        data.Headers,
	}
	return response, nil
}
