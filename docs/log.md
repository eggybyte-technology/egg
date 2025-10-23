# 🧩 EggyByte Logging Design

---

## 🎯 设计目标

| 目标                   | 描述                                          |
| -------------------- | ------------------------------------------- |
| ✅ **严格符合 logfmt 规范** | 每个 `key=value` 对独立，所有特殊字符必须转义               |
| ✅ **单行输出**           | 确保 Loki、Promtail 一行一条日志                     |
| ✅ **彩色等级输出**         | 控制台中高可读，Loki 收集时可安全去色                       |
| ✅ **简单、可预测、机器可解析**   | 可被任何 logfmt 解析器（Go kit、Loki、Datadog）正确读取    |
| ✅ **无语义冗余**          | 不输出 namespace、service、timestamp 等 Loki 自动字段 |

---

## 🧠 logfmt 基础规范（EggyByte 采用版）

**每条日志是若干 `key=value` 对，用空格分隔：**

```
key=value key2=value2 key3=value3
```

**规则：**

1. **键名** 只能包含 `[a-zA-Z0-9_-]`
2. **值** 必须满足：

   * 若为字符串：

     * 无空格、无等号、无引号 → 原样输出
       ✅ `user=alice`
     * 含空格或特殊字符 → 用双引号包裹，并转义引号
       ✅ `msg="cache miss on key=\"user:123\""`
   * 若为布尔或数字 → 原样输出
     ✅ `count=3 duration_ms=24`
3. **字段顺序** 建议固定：

   ```
   level → msg → context/fields (lexicographically sorted)
   ```

---

## 🎨 ANSI 颜色规范（仅 level 部分带色）

| Level | Color Code                     | 含义     |
| ----- | ------------------------------ | ------ |
| DEBUG | `\033[36m` (Cyan)              | 调试信息   |
| INFO  | `\033[32m` (Green)             | 正常状态   |
| WARN  | `\033[33m` (Yellow)            | 潜在风险   |
| ERROR | `\033[31m` (Red)               | 错误事件   |
| FATAL | `\033[41;97m` (Red background) | 致命错误   |
| RESET | `\033[0m`                      | 恢复默认颜色 |

---

## 🧩 日志输出格式（单行、严格 logfmt）

```
level=<colored-level> msg="<message>" key1=value1 key2="value 2" key3="contains=equals"
```

---

## 🧱 字段规范

| 字段名         | 类型     | 描述                                |
| ----------- | ------ | --------------------------------- |
| `level`     | string | 日志等级，带颜色输出（控制台模式）                 |
| `msg`       | string | 主消息，始终被引号包裹                       |
| `trace_id`  | string | 可选，分布式追踪 ID                       |
| `error`     | string | 可选，错误描述（带引号）                      |
| `component` | string | 可选，组件名或子系统名                       |
| 其他字段        | mixed  | 自定义上下文字段，统一采用 logfmt key=value 形式 |

---

## 💬 示例

### ✅ INFO 日志

```
\033[32mlevel=INFO\033[0m msg="Configuration loaded successfully" config_file="/etc/yao/config.yaml" environment=prod
```

### ⚠️ WARN 日志

```
\033[33mlevel=WARN\033[0m msg="Slow request detected" trace_id=8c9f node=proxy-1 duration_ms=1248
```

### ❌ ERROR 日志（带等号、空格）

```
\033[31mlevel=ERROR\033[0m msg="Failed to connect to TiDB cluster" trace_id=9c2b8 node=node-2 error="connection timeout" endpoint="tidb-cluster:4000"
```

### 💀 FATAL 日志

```
\033[41;97mlevel=FATAL\033[0m msg="Startup failure: missing secret key" component=bootstrap
```

---

## ⚙️ 格式设计细节

| 规则                   | 说明                     |
| -------------------- | ---------------------- |
| **所有字符串字段必须双引号包裹**   | 保证即使有空格或等号也安全可解析       |
| **数字与布尔值不加引号**       | 与标准 logfmt 保持一致        |
| **字段之间单空格分隔**        | Loki / Promtail 解析最佳实践 |
| **颜色仅作用于 level=XXX** | 避免破坏后续字段结构             |
| **每条日志严格单行**         | 不允许 `\n`，保证容器日志一行一条    |
| **禁止 emoji**         | 避免 Unicode 宽度差异导致混乱    |
| **字段排序**             | 固定顺序提高一致性与差异检测         |
| **escape 引号规则**      | `"` → `\"`；`\` → `\\`  |

---

## 🧱 控制台与 Loki 输出差异

| 模式                 | 彩色  | ANSI 编码       | Loki 解析安全性     |
| ------------------ | --- | ------------- | -------------- |
| **本地开发**           | ✅ 是 | 保留 ANSI       | 可读性最强          |
| **Kubernetes 容器**  | ✅ 是 | Promtail 自动剥离 | Loki 自动解析字段    |
| **日志存储 / Loki 查询** | ❌ 否 | 纯文本（无颜色）      | 100% logfmt 兼容 |

> 💡 Loki 的 Promtail 在 pipeline_stages 中执行 `cri` 后会移除 ANSI 颜色序列，
> 不会影响字段解析。

---

## 🧰 推荐的字段最小集合

| 场景     | 推荐字段                                                        |
| ------ | ----------------------------------------------------------- |
| 普通运行日志 | `level`, `msg`, `component`, `trace_id`                     |
| 请求日志   | `level`, `msg`, `trace_id`, `duration_ms`, `method`, `path` |
| 错误日志   | `level`, `msg`, `trace_id`, `error`, `endpoint`, `retries`  |
| 初始化日志  | `level`, `msg`, `config_file`, `environment`                |

---

## 🧱 输出规范总结

| 项目                            | 规范                                     |
| ----------------------------- | -------------------------------------- |
| **输出格式**                      | 严格 logfmt (`key=value key2="value 2"`) |
| **每行一条日志**                    | ✅                                      |
| **颜色应用**                      | 仅作用于 level                             |
| **字符串字段**                     | 一律双引号包裹                                |
| **字段顺序**                      | level → msg → 其他键按字典序                  |
| **时间戳 / service / namespace** | ❌ 不输出                                  |
| **emoji**                     | ❌ 禁止                                   |
| **K8s 兼容**                    | ✅ 可被 Promtail+Loki 自动解析                |

---

## ✅ 标准示例总结

| Level | 示例输出                                                                                                                       |
| ----- | -------------------------------------------------------------------------------------------------------------------------- |
| INFO  | `\033[32mlevel=INFO\033[0m msg="Proxy started" port=8080 ready=true`                                                       |
| WARN  | `\033[33mlevel=WARN\033[0m msg="Cache approaching limit" usage_pct=92`                                                     |
| ERROR | `\033[31mlevel=ERROR\033[0m msg="Request failed" trace_id=8b12 error="context deadline exceeded" endpoint="/v1/cache/get"` |
| FATAL | `\033[41;97mlevel=FATAL\033[0m msg="Startup failure: missing secret key" component="bootstrap"`                            |

---

## 💡 Implementation Guidelines（概要）

1. 所有字段通过安全转义函数进行处理：

   * 字符串含空格、等号、引号 → 自动加引号并转义。
2. 输出器统一封装为：

   * `logx.Info(msg string, fields ...Fields)`
   * `logx.Error(msg string, fields ...Fields)`
3. 控制台模式自动彩色输出（仅 level 字段彩色）。
4. 确保 `fmt.Fprintln(os.Stdout, line)` 输出单行。
5. 支持通过环境变量控制：

   * `LOG_COLOR=true`
   * `LOG_LEVEL=INFO`
   * `LOG_FORMAT=logfmt`

---

## 🧭 最终总结

> **EggyByte 日志设计标准：**
>
> * 格式：**严格单行 logfmt**
> * 语义：`level= msg= ...`
> * 彩色：仅 `level` 带色
> * 安全：所有字符串自动加引号与转义
> * 可解析：Promtail `logfmt` stage 可直接识别
> * 通用：适配 CLI、微服务、Kubernetes、Loki 全栈环境