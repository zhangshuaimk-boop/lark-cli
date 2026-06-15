# appsmock demo —— 把 apps 域请求转发到本地 mock

一个自包含的 demo，用来验证 apps 域鉴权调研结论中提到的**「fork 极低入侵」**方案：

- `extension/credential/appsmock/`  —— 桩凭证 provider（build tag `appsmock`）
- `extension/transport/appsmock/`   —— URL 重写 interceptor（build tag `appsmock`）
- `main_appsmock.go`                 —— 通过空白 import 激活上面两个扩展（build tag `appsmock`）
- `demo/cmd/mockserver/`             —— 本地 mock HTTP 服务，返回模拟的 apps OpenAPI 响应
- `demo/e2e.sh`                       —— 端到端测试脚本（正反向各一例）

## 运行方式

在仓库根目录执行：

```bash
bash demo/e2e.sh
```

预期末尾输出：

```
[4/5] running positive case: apps +list
  positive case OK
[5/5] running negative case: contact +get-user must NOT hit mock
  negative case OK (mock log empty)

ALL E2E CHECKS PASSED
```

## 手动玩

```bash
go build -tags appsmock -o /tmp/lark-cli-appsmock .
go build -o /tmp/mockserver ./demo/cmd/mockserver

/tmp/mockserver -addr 127.0.0.1:7878 &

LARK_CLI_APPS_MOCK=http://127.0.0.1:7878 \
  /tmp/lark-cli-appsmock apps +list --format json
# → 返回 mock 服务里写死的 mock_app_aaa / mock_app_bbb

LARK_CLI_APPS_MOCK=http://127.0.0.1:7878 \
  /tmp/lark-cli-appsmock contact +get-user --user-id u_x --user-id-type open_id
# → 不会被路由到 mock（mock 这次请求的日志保持空白）
```

## 激活机制（双重开关）

两道独立闸门，**都开**才生效：

1. **Build tag `appsmock`**：不带 tag 时，appsmock 包根本不会被编译进二进制。
   默认 `go build` 产出的 binary 与 upstream 行为完全一致。
2. **环境变量 `LARK_CLI_APPS_MOCK`**：带 tag 的 binary 在没有设置该环境变量时
   也什么都不做。环境变量未设置时，凭证 provider 和 transport interceptor 的
   `init()` 都会提前 return、不注册自己，原有的凭证/transport 行为继续生效。

## 这份 demo 证明了 fork 方案的哪些断言

| 方案断言 | 本 demo 怎么证明的 |
|---|---|
| 对 `internal/` 零侵入 | `git status` 显示改动只在 `extension/`、`main_appsmock.go`、`demo/`，`internal/` 一行未改 |
| 通过现有扩展点一行注册 | 全部插桩就是 `credential.Register(&Provider{})` + `exttransport.Register(&Provider{})` |
| apps 域可以从 `req.URL.Path` 识别 | `IsAppsDomain("/open-apis/spark/...")` 跑通；单测覆盖正反向 |
| 其他域不受影响 | 反向 e2e 用例断言：调非 apps 命令时 mock 日志为空 |
| upstream 同步摩擦 ≈ 0 | 全是新增文件，未编辑任何 upstream 既有文件 |

## 越过 demo、走向落地的下一步

- 把 dummy token 替换成真正的内部鉴权 client（HMAC / mTLS / 内网签名 …），逻辑塞进
  credential provider 的 `ResolveToken`。
- 决定 interceptor 用哪种策略：当前用「重写 URL」（mock server 直接讲 `/open-apis/spark/...` 这套路径），
  也可以改用 `AbortableInterceptor.PreRoundTripE` 在前置 hook 里短路、合成一个 `*http.Response` 返回
  （不走真实 HTTP 出网，适合内部鉴权走非 HTTP 协议的场景）。
- 若需要和 auth sidecar（`-tags authsidecar`）共存：`exttransport.Register` 是 last-write-wins，
  需要写一个组合 provider，apps 域走内部鉴权、其他域走 sidecar。当前两个 build tag 设计上互斥
  （各自 `Register` 一个 Provider 即覆盖前者）。
- `AppsDomainPrefixes` 列表需要随 upstream 演进同步更新。建议加一个 CI 检查：
  `grep -h '/open-apis/spark' shortcuts/apps/*.go` 跟列表 diff，发生漂移时报警。
