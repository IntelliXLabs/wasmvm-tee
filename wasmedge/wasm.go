package wasmedge

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

type host struct {
	fetchResult []byte
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
func (h *host) fetch(_ interface{}, callframe *wasmedge.CallingFrame, params []interface{}) ([]interface{}, wasmedge.Result) {
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

	return []interface{}{interface{}(len(respBody))}, wasmedge.Result_Success
}

// Host function for writting memory
func (h *host) writeMem(_ interface{}, callframe *wasmedge.CallingFrame, params []interface{}) ([]interface{}, wasmedge.Result) {
	// write source code to memory
	pointer := params[0].(int32)
	mem := callframe.GetMemoryByIndex(0)
	mem.SetData(h.fetchResult, uint(pointer), uint(len(h.fetchResult)))

	return nil, wasmedge.Result_Success
}

func ExecuteWasm(wasmCode []byte, fnName string, params []any) ([]any, error) {
	fmt.Println("Go: Args:", os.Args)

	wasmedge.SetLogErrorLevel()

	conf := wasmedge.NewConfigure(wasmedge.WASI)
	vm := wasmedge.NewVMWithConfig(conf)
	obj := wasmedge.NewModule("env")

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

	r, _ := vm.Execute(fnName, params...)
	fmt.Printf("There are %d 'google' in source code of google.com\n", r[0])

	obj.Release()
	vm.Release()
	conf.Release()

	return r, nil
}
