# openclaw-weixin-go

[English README](./README.en.md)

`openclaw-weixin-go` 是一个面向 Go 开发者的微信 iLink 协议 SDK，专注于把扫码登录、长轮询收发消息和默认本地存储封装成一套可直接复用的纯 Go 能力。

核心能力：

- 扫码登录与二维码状态轮询
- `getupdates` 长轮询收消息
- 文本消息发送与 typing 指示
- 默认文件存储：`account.json`、`ctx_tokens.json`、`sync_buf.txt`
- 一个可直接使用的 CLI：`login`、`whoami`、`poll`、`send`、`logout`

作者：`jtai团队（曾能混&tang先森）`  
邮箱：`jwhna1@gmil.com`  
官网：[jtai.cc](https://jtai.cc)

## 项目定位

这个仓库不是腾讯官方 Go SDK。

协议实现与字段行为参考 [Tencent/openclaw-weixin](https://github.com/Tencent/openclaw-weixin)，目标是为 Go 生态提供一个更容易集成、便于二次封装、适合独立项目直接接入的微信协议 SDK。

## 目录结构

- `client/`: iLink 协议客户端与 DTO
- `store/`: 默认文件存储实现
- `cmd/openclaw-weixin-go/`: CLI 入口
- `examples/echo-bot/`: 最小轮询示例

## 安装

```bash
go build ./cmd/openclaw-weixin-go
```

## CLI 命令

```bash
openclaw-weixin-go login  --data-dir ./data
openclaw-weixin-go whoami --data-dir ./data
openclaw-weixin-go poll   --data-dir ./data
openclaw-weixin-go send   --data-dir ./data --to wxid_xxx --text "hello"
openclaw-weixin-go logout --data-dir ./data
```

`login` 会在终端中直接渲染二维码，同时输出二维码 URL 和轮询状态，并在成功后把登录信息保存到本地目录。

## Quick Start

推荐先按下面顺序完成一轮最小联调：

```bash
go run ./cmd/openclaw-weixin-go login --data-dir ./data
go run ./cmd/openclaw-weixin-go whoami --data-dir ./data
go run ./cmd/openclaw-weixin-go poll --data-dir ./data
```

说明：

- `login`：终端直接显示可扫码二维码，并显示 `waiting for scan`、`scanned, please confirm on your phone`、`login confirmed` 等状态
- 登录成功后，账号信息默认保存到 `./data/wechat/account.json`
- `whoami`：检查 `account.json` 是否已经成功落盘
- `poll`：验证 `getupdates` 长轮询是否已经通畅

下面是一张脱敏后的 CLI 登录最小测试截图：

![CLI login demo](docs/images/cli-login-demo.png)

## SDK 用法

```go
package main

import (
    "context"

    "github.com/jwhna1/openclaw-weixin-go/client"
    "github.com/jwhna1/openclaw-weixin-go/store"
)

func main() {
    st := store.NewFileStore("./data")
    acct, _ := st.LoadAccount()

    cli := client.New(client.Options{
        BaseURL:        acct.BaseURL,
        ClientIDPrefix: "my_bot",
    })

    resp, _ := cli.GetUpdates(context.Background(), acct.Token, "", client.DefaultGetUpdatesTimeout)
    _ = resp
}
```

## 当前支持范围

已覆盖：

- 扫码登录
- 文本消息发送
- 长轮询收消息
- `context_token` 持久化
- `sync_buf` 持久化

暂不承诺：

- 媒体上传下载
- 多账号高级调度
- 完整业务网关适配

## 开源与免责

- 请避免把本仓库描述为腾讯官方 SDK。
- 本仓库采用 `MIT` 协议发布。
- 软件按 `AS IS` 提供，不附带任何明示或暗示担保；因使用、修改或分发本项目产生的风险与责任由使用方自行承担。
- 协议字段优先以当前实测行为为准，而不是未验证的旧文档描述。

## 相关资源

- [OpenClaw 官方文档](https://docs.openclaw.ai/)
- [腾讯官方 npm 包 `@tencent-weixin/openclaw-weixin`](https://www.npmjs.com/package/@tencent-weixin/openclaw-weixin)
