# api-client-go

## gRPC 错误传递

公共 `errors.ToGRPC` 使用 gRPC 状态与标记为 `gts.ResponseError` 的 ErrorInfo 传递结构化错误，保留业务 code、HTTP 状态、消息、字段及可 JSON 序列化的模板数据。普通未知错误保持 Unknown，取消/超时及原生 gRPC 状态保持原语义。不会把网络、限流或服务错误一律转换为 400。

`conn.CreateConn` 为所有通过它创建的服务连接安装统一解码拦截器；`local_grpc.Server.Invoke` 使用同一编码/解码路径。调用者可用 `errors.As` 提取 `github.com/felix-186/errors.ResponseError`。直接自行创建 gRPC 连接的调用者，应安装 `errors.UnaryClientErrorDecoder` 或调用 `errors.FromGRPC`，否则仍可读取 gRPC 标准状态与 ErrorInfo。已有 `api.Response{status:false}` 解析继续兼容，不修改 protobuf 字段。

需先发布并升级客户端，再升级使用新公共错误中间件的服务端。旧客户端可看到正确的标准 gRPC 状态，但不能自动还原业务码；旧服务端已丢失成 Unknown 的业务码无法在客户端凭空恢复。未在本次修改中发布版本或变更各服务的版本依赖。详情测试包括 `go test ./errors ./local_grpc ./conn`，以及 service 公共中间件中的真实 gRPC 往返测试。

`api-client-go` 是一个面向 IOT 平台的 Go 客户端库，用于统一访问平台内多个 gRPC 服务，并封装认证、服务发现、配置加载与本地调试能力。

当前模块名：

```go
module github.com/felix-186/api-client-go
```

## 功能概览

- 统一聚合多个业务客户端到一个 `Client`
- 基于 `AK/SK` 自动获取并刷新访问令牌
- 支持通过 etcd + Kratos registry 发现服务
- 支持从配置中心加载配置并合并到本地参数
- 支持 `LiteMode` 跳过注册中心能力
- 支持本地 `local_grpc` 方式进行调试或测试
- 提供常用业务访问入口，例如：
  - `AuthClient`
  - `SpmClient`
  - `CoreClient`
  - `FlowClient`
  - `WarningClient`
  - `DriverClient`
  - `DataServiceClient`
  - `FlowEngineClient`
  - `ReportClient`
  - `AlgorithmClient`
  - `DataRelayClient`
  - `JsServerClient`
  - `SyncClient`
  - `ComputeRecordClient`
  - `AIClient`

主入口定义见 `client.go:42`。

## 安装

```bash
go get github.com/felix-186/api-client-go
```

## 依赖要求

通常需要以下基础设施之一：

### 1. 标准 gRPC / 注册中心模式

适用于实际接入平台服务：

- etcd
- Kratos registry
- 有效的 `AK/SK`

### 2. 本地模式

适用于测试或嵌入式本地调用：

- 本地构造 `local_grpc.Server`
- 使用 `NewLocalClient(...)`

## 快速开始

### 创建 etcd 客户端

```go
package main

import (
    "log"
    "time"

    api_client_go "github.com/felix-186/api-client-go"
    "github.com/felix-186/api-client-go/config"
    clientv3 "go.etcd.io/etcd/client/v3"
    "google.golang.org/grpc"
)

func main() {
    etcdCli, err := clientv3.New(clientv3.Config{
        Endpoints:   []string{"127.0.0.1:2379"},
        DialTimeout: 60 * time.Second,
        DialOptions: []grpc.DialOption{grpc.WithBlock()},
    })
    if err != nil {
        log.Fatal(err)
    }
    defer etcdCli.Close()

    cli, clean, err := api_client_go.NewClient(etcdCli, config.Config{
        LiteMode:   false,
        EtcdConfig: "/config/dev.json",
        Metadata: map[string]string{
            "env": "aliyun",
        },
        Type:      config.Tenant,
        AK:        "your-ak",
        SK:        "your-sk",
        Timeout:   60,
        KeepAlive: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer clean()

    _ = cli
}
```

`NewClient(...)` 定义见 `client.go:67`。

## 配置说明

配置结构定义见 `config/config.go:12`。

```go
config.Config{
    LiteMode:   false,
    Gateway:    "",
    GatewayGrpc:"127.0.0.1:9224",
    EtcdConfig: "/config/dev.json",
    Metadata: map[string]string{
        "env": "aliyun",
    },
    Services: map[string]config.Service{
        // "core": {Metadata: map[string]string{"env": "local"}},
    },
    Type:      config.Tenant, // 或 config.Project
    ProjectId: "",
    AK:        "your-ak",
    SK:        "your-sk",
    Timeout:   60,
}
```

### 关键字段

- `LiteMode`：为 `true` 时不走配置中心 / 注册中心完整逻辑
- `EtcdConfig`：配置中心路径，如 `/config/dev.json`
- `Metadata["env"]`：用于服务发现时筛选环境
- `Services`：可为某个具体服务单独指定 metadata 或直连地址
- `Type`：认证类型，支持：
  - `config.Tenant`
  - `config.Project`
- `ProjectId`：当 `Type == config.Project` 时使用
- `AK` / `SK`：平台认证凭据
- `Timeout`：请求超时时间

## 认证机制

认证客户端定义见 `auth/auth_client.go:24`。

库会通过 `AK/SK` 自动获取 token，并在过期前按 `ExpirePrecision` 提前刷新。

你也可以直接获取当前 token：

```go
token, err := cli.GetToken()
```

实现见 `auth.go:3`。

## 常见用法

### 查询项目

```go
package main

import "context"

func example(cli *api_client_go.Client) error {
    var result []map[string]interface{}
    return cli.QueryProject(context.Background(), map[string]interface{}{}, &result)
}
```

相关测试示例见 `test/client_test.go:182`。

### 查询表结构

```go
package main

import "context"

func example(cli *api_client_go.Client, projectID string) error {
    var result []map[string]interface{}
    return cli.QueryTableSchema(context.Background(), projectID, map[string]interface{}{}, &result)
}
```

相关示例见 `client_test.go:126`、`test/client_test.go:130`。

### 运行流程

```go
package main

import "context"

func example(cli *api_client_go.Client, projectID string, flowConfig string, element []byte, vars map[string]interface{}) error {
    _, err := cli.Run(context.Background(), projectID, flowConfig, element, vars)
    return err
}
```

相关示例见 `client_test.go:58`。

## 本地调试模式

如果你已经在进程内注册了若干 gRPC service，可使用本地模式：

```go
ss := local_grpc.NewServer()

cli, clean, err := api_client_go.NewLocalClient(cfg, ss)
if err != nil {
    return err
}
defer clean()
```

- 本地 server 定义见 `local_grpc/server.go:40`
- 本地客户端入口见 `client.go:71`

这种方式适合：

- 单元测试
- 本地联调
- 无需真实网络连接的服务调用模拟

## 服务可用性检查

聚合客户端还提供了服务缓存与服务存在性检查能力：

```go
ok, err := cli.Service.IsExist(ctx, "js-server")
```

实现见 `service.go:29`。

## 项目结构

主要目录说明：

- `config/`：配置结构定义
- `auth/`：认证与凭据处理
- `local_grpc/`：本地 gRPC 调试实现
- `apicontext/`：请求上下文辅助封装
- `errors/`：平台响应错误解析
- `ai/`、`core/`、`flow/`、`driver/`、`report/` 等：各服务 proto 与客户端封装
- `test/`：集成测试与使用示例

## 注意事项

- 当前仓库包含大量由 proto 生成的 `*.pb.go` 和 `*_grpc.pb.go` 文件
- 部分测试依赖真实环境、真实 etcd、真实服务地址和有效凭据，不能直接在纯本地环境运行
- `test/client_test.go` 中存在演示性质的环境配置，实际使用时请替换为你自己的地址和凭据

## 参考代码位置

- 聚合客户端：`client.go:42`
- 标准客户端初始化：`client.go:67`
- 本地客户端初始化：`client.go:71`
- 配置结构：`config/config.go:12`
- 认证逻辑：`auth/auth_client.go:24`
- 本地 gRPC server：`local_grpc/server.go:40`
- 服务检查：`service.go:13`

## License

如果你需要开源发布，建议补充明确的 License 文件；当前仓库根目录尚未看到独立 license 说明。
