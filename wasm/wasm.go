package wasm

import (
	"fmt"
	"io"
	"net/http"

	"github.com/second-state/WasmEdge-go/wasmedge"
	bindgen "github.com/second-state/wasmedge-bindgen/host/go"
)

type host struct {
	fetchResult []byte
}

// ExecuteWasm executes WebAssembly code and returns proto Value structures
func ExecuteWasm(wasmCode []byte, fnName string, params []any) ([]any, error) {
	wasmedge.SetLogErrorLevel()

	conf := wasmedge.NewConfigure(wasmedge.WASI)
	defer conf.Release()

	vm := wasmedge.NewVMWithConfig(conf)
	defer vm.Release()

	obj := wasmedge.NewModule("env")
	defer obj.Release()

	h := host{}
	// Add host functions into the module instance
	funcFetchType := wasmedge.NewFunctionType(
		[]*wasmedge.ValType{
			wasmedge.NewValTypeI32(),
			wasmedge.NewValTypeI32(),
		},
		[]*wasmedge.ValType{
			wasmedge.NewValTypeI32(),
		})

	hostFetch := wasmedge.NewFunction(funcFetchType, h.fetch, nil, 0)
	obj.AddFunction("fetch", hostFetch)

	// Add HTTP link function - supports full HTTP requests with JSON
	funcHttpType := wasmedge.NewFunctionType(
		[]*wasmedge.ValType{
			wasmedge.NewValTypeI32(),
			wasmedge.NewValTypeI32(),
		},
		[]*wasmedge.ValType{
			wasmedge.NewValTypeI32(),
		})
	hostHttp := wasmedge.NewFunction(funcHttpType, h.http, nil, 0)
	obj.AddFunction("http", hostHttp)

	funcWriteType := wasmedge.NewFunctionType(
		[]*wasmedge.ValType{
			wasmedge.NewValTypeI32(),
		},
		[]*wasmedge.ValType{})
	hostWrite := wasmedge.NewFunction(funcWriteType, h.writeMem, nil, 0)
	obj.AddFunction("write_mem", hostWrite)

	vm.RegisterModule(obj)

	vm.LoadWasmBuffer(wasmCode)
	vm.Validate()
	vm.Instantiate()

	bg := bindgen.New(vm)
	bg.Instantiate()
	// Execute WASM function
	results, _, err := bg.Execute(fnName, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute WASM function: %v", err)
	}

	return results, nil
}

// do the http fetch
func fetch(url string) []byte {
	resp, err := http.Get(string(url))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return body
}

// Host function for fetching
func (h *host) fetch(_ any, callframe *wasmedge.CallingFrame, params []any) ([]any, wasmedge.Result) {
	// get url from memory
	pointer := params[0].(int32)
	size := params[1].(int32)
	mem := callframe.GetMemoryByIndex(0)
	data, _ := mem.GetData(uint(pointer), uint(size))
	url := make([]byte, size)

	copy(url, data)

	respBody := fetch(string(url))

	if respBody == nil {
		return nil, wasmedge.Result_Fail
	}

	// store the source code
	h.fetchResult = respBody

	return []any{any(len(respBody))}, wasmedge.Result_Success
}

// Host function for writting memory
func (h *host) writeMem(_ any, callframe *wasmedge.CallingFrame, params []any) ([]any, wasmedge.Result) {
	// write source code to memory
	pointer := params[0].(int32)
	mem := callframe.GetMemoryByIndex(0)
	mem.SetData(h.fetchResult, uint(pointer), uint(len(h.fetchResult)))

	return nil, wasmedge.Result_Success
}
