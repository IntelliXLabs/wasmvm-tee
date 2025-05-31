# WASMVM-TEE WasmEdge Integration Changes

## 概述

本项目已从基于 DTVM 的实现转换为基于 WasmEdge 虚拟机的实现，同时添加了 WASI 支持并将所有返回值转换为 protobuf Value 结构。

## 主要修改

### 1. WasmEdge 集成 (`wasmedge/wasm.go`)

- **新增功能**：完整的 WasmEdge 集成，支持 WASI
- **类型转换**：将 WasmEdge 返回值自动转换为 proto `Value` 结构
- **支持的类型**：
  - `int32`, `int64`, `int`, `uint32`, `uint64`, `uint`
  - `float32`, `float64`
  - `wasmedge.V128` (报错，因为 proto 中不支持)

#### 关键函数

```go
func ExecuteWasm(wasmCode []byte, fnName string, params []any) ([]*pb.Value, error)
func convertToProtoValue(result any) (*pb.Value, error)
func convertResults(results []any) ([]*pb.Value, error)
```

### 2. 服务器实现更新 (`dtvm/server.go`)

- **移除依赖**：不再依赖 `dtvm-go` 库
- **新增 WasmEdge 调用**：直接在服务器中集成 WasmEdge 执行逻辑
- **类型安全**：所有类型转换都包含边界检查和错误处理

#### 主要变更

```go
// 旧实现：使用 DTVM
func (s *Server) executeWASMFunction(config *DTVMRuntimeConfig, bytecode []byte, fnName string, inputs []string) ([]*Value, error)

// 新实现：使用 WasmEdge
func (s *Server) executeWasmEdge(wasmCode []byte, fnName string, params []any) ([]*Value, error)
```

### 3. 测试更新 (`wasmedge/wasm_test.go`)

- **更新测试**：适配新的返回类型 `[]*pb.Value`
- **类型检查**：添加了完整的类型断言测试

### 4. Proto Value 转换

#### 支持的映射关系

| WasmEdge 类型 | Proto 类型 | 说明 |
|---------------|------------|------|
| `int32` | `VALUE_TYPE_INT32` | 直接映射 |
| `int64` | `VALUE_TYPE_INT64` | 直接映射 |
| `int` | `VALUE_TYPE_INT32` 或 `VALUE_TYPE_INT64` | 根据平台位数决定 |
| `uint32` | `VALUE_TYPE_INT32` | 检查溢出 |
| `uint64` | `VALUE_TYPE_INT64` | 检查溢出 |
| `uint` | `VALUE_TYPE_INT32` 或 `VALUE_TYPE_INT64` | 根据平台位数决定，检查溢出 |
| `float32` | `VALUE_TYPE_FLOAT32` | 直接映射 |
| `float64` | `VALUE_TYPE_FLOAT64` | 直接映射 |
| `wasmedge.V128` | 错误 | 不支持向量类型 |

#### 类型转换函数

```go
func convertToProtoValue(result any) (*Value, error) {
    switch v := result.(type) {
    case int32:
        return &Value{
            Type: ValueType_VALUE_TYPE_INT32,
            Value: &Value_Int32Value{Int32Value: v},
        }, nil
    // ... 其他类型
    }
}
```

## 安全特性

### 1. 溢出检查

```go
case uint32:
    if v > math.MaxInt32 {
        return nil, fmt.Errorf("uint32 值 %d 超出 int32 范围", v)
    }
```

### 2. 平台兼容性

```go
case int:
    if unsafe.Sizeof(v) == 4 {
        // 32位平台处理
    } else {
        // 64位平台处理
    }
```

### 3. 错误处理

- 所有类型转换都包含完整的错误处理
- 不支持的类型会返回明确的错误信息
- 资源自动清理（defer 语句）

## 使用示例

### 基本用法

```go
// 加载 WASM 字节码
wasmBytes, err := os.ReadFile("example.wasm")
if err != nil {
    log.Fatal(err)
}

// 执行函数
results, err := wasmedge.ExecuteWasm(wasmBytes, "add", []any{int32(10), int32(20)})
if err != nil {
    log.Fatal(err)
}

// 处理结果
for i, result := range results {
    switch result.Type {
    case pb.ValueType_VALUE_TYPE_INT32:
        fmt.Printf("结果[%d]: int32 = %d\n", i, result.GetInt32Value())
    case pb.ValueType_VALUE_TYPE_INT64:
        fmt.Printf("结果[%d]: int64 = %d\n", i, result.GetInt64Value())
    // ... 其他类型
    }
}
```

### gRPC 服务使用

服务器现在自动处理类型转换，客户端只需要发送标准的 `WASMVMExecutionRequest`：

```go
req := &dtvm.WASMVMExecutionRequest{
    Execution: &dtvm.WASMVMExecution{
        Bytecode: base64.StdEncoding.EncodeToString(wasmBytes),
        FnName:   "add",
        Inputs:   []string{"10", "20"},
    },
}

response, err := client.Execute(ctx, req)
```

## 兼容性说明

### 保持兼容

- gRPC 接口保持不变
- Proto 消息格式保持不变
- 客户端无需修改

### 行为变更

- 底层执行引擎从 DTVM 改为 WasmEdge
- 添加了 WASI 支持
- 类型转换更加严格和安全

## 依赖更新

### 新增依赖

```go
import "github.com/second-state/WasmEdge-go/wasmedge"
import "math"
import "unsafe"
```

### 移除依赖

```go
// 移除：import "github.com/IntelliXLabs/dtvm-go"
```

## 文件变更摘要

- ✅ `wasmedge/wasm.go` - 完全重写，添加 WasmEdge 集成和类型转换
- ✅ `wasmedge/wasm_test.go` - 更新测试以适配新返回类型
- ✅ `dtvm/server.go` - 重构服务器实现，移除 DTVM 依赖
- ✅ `examples/wasmedge_example.go` - 新增使用示例
- ✅ `WASMVM_CHANGES.md` - 本文档

这些修改确保了项目能够使用高性能的 WasmEdge 运行时，同时保持与现有客户端的完全兼容性。
