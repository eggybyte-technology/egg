对，你已经踩到这个痛点的本质了：**在多模块 monorepo 的开发阶段，`go mod tidy` 不是一个“随便按就行的一键同步器”。**
我们需要把“依赖管理”拆成两类来看：**内部模块依赖（core / runtimex / connectx 这种你自己写的）**，和**外部依赖（第三方开源库）**。它们的管理方式其实不一样。

我会给你一套“开发阶段的依赖管理准则”。照着这套做，整个库在开发期会稳定、可协作、不会乱写奇怪版本。

---

## 最重要的结论（先说答案）

1. **内部模块（同仓库的 core/runtimex/...）在开发阶段不靠 `go mod tidy` 来管理。靠 `go work`。**

   * 也就是：工作区解决“我本地怎么跑”，而不是 go.mod 解决。
   * go.mod 在开发期可以是“过时的依赖版本”，只要 build/test 能跑就行。

2. **外部依赖（例如 `github.com/bufbuild/connect-go`、`go.uber.org/zap`）的版本升级，请用 `go get -u` / `go get <module>@<version>` 来显式控制。**

   * 然后再 `go mod tidy` 来清理无用依赖、多余的 sum。
   * 也就是说：`tidy` 是清扫工具，而不是拉依赖的主工具。

3. “日常写代码时每次保存文件都随手 tidy”是错误模式。
   发布前 / 大清理时再 tidy，而不是每 commit 一次 tidy 一次。

下面详细展开，并给出具体操作手册。

---

## 一类依赖：内部模块（core / runtimex / connectx ...）

问题：这些模块互相 import，但还没发布新 tag；此时 `go mod tidy` 会试图从远程找版本，引发脏版本或 network 拉取，对吧。

解决方式是 **把“本地能互相引用”交给 `go work`，而不是强行让每个 go.mod 永远是最新的**。

### 标准做法（开发期）

* 仓库根目录维护一个公共 `go.work`，列出所有内部模块：

  ```bash
  go work init ./core
  go work use ./runtimex ./connectx ./k8sx ./obsx ./cli
  ```

  这个 `go.work` 是**必须进 git**的，让每个人拉仓库后都能本地联动 build。

* 之后，无论你在 `connectx` 里 import `github.com/eggybyte-technology/egg/runtimex/app`,
  `go build ./...` / `go test ./...` 都会走工作区里的本地源码。

* 这意味着：即使 `connectx/go.mod` 里当前写的还是

  ```go
  require github.com/eggybyte-technology/egg/runtimex v0.2.0
  ```

  你依然可以测试依赖“下一版 runtimex 的未发布改动”，因为 `go work` 会优先用本地副本覆盖掉 v0.2.0。

→ 换句话说：**在开发期，go.mod 里的版本不需要跟着你每次本地修改一起变。它只是“上一次发布的已知好版本号”，而不是“我现在工作拷贝的 HEAD”。**

非常关键：这就是为什么我们说“不要指望 tidy 来负责内部依赖”。

### 那什么时候要更新 go.mod 里的内部依赖版本？

只在 **准备发版** 的时候。具体节奏：

1. 先给底层模块（core）发 tag，比如 `core/v0.3.2`。
2. 在更高层模块（runtimex）里，手动把 `require github.com/eggybyte-technology/egg/core` 改成 `v0.3.2`。
3. 现在在 `runtimex/` 里运行：

   ```bash
   go mod tidy
   ```

   这一步是“我要准备发布 runtimex 了”。
   tidy 现在是合理的，因为它会把 `core@v0.3.2` 落实成干净的 go.mod/go.sum（可被任何下游项目重现）。

总结一句：
**内部依赖的 go.mod 版本号 = “我上一次对外承诺过的 tag”**，
**而不是 = “我本地正在写的新东西”。**
本地新东西靠 `go.work` 解耦，而不是靠 go.mod+tidy。

---

## 二类依赖：外部模块（第三方开源包）

这一类就简单多了：这部分你确实应该通过 `go get` / `go mod tidy` 来管理，但要分工明确。

### 推荐流程

假设你想把 `connect-go` 升级到新版本，或者你新 import 了某个外部库。

1. 在对应子模块目录下（比如 `connectx/`）执行显式升级：

   ```bash
   go get github.com/bufbuild/connect-go@v1.16.0
   ```

   或者全量小升级：

   ```bash
   go get -u ./...
   ```

   （如果你担心破坏，可以不加 `-u=patch` 这种策略化，就手动指定版本。）

2. 然后再跑：

   ```bash
   go mod tidy
   ```

`go get` 做版本决策（我要哪个版本），
`go mod tidy` 做卫生（删没用的、补 go.sum、把图自洽）。

这是安全的，因为外部依赖“本来就该来自远程 tag”，让 tidy 去访问远程是合理的。

---

## 所以，开发阶段到底该怎么活？

可以直接当成操作规程抄到团队 README：

### 开发日常（写代码 / 调试阶段）

1. **永远在仓库根目录使用同一个 `go.work`，并把它提交到 Git。**
2. 想同时改 `core` 和 `runtimex`？可以，直接改。用 `go build` / `go test` 在 workspace 下验证。
3. **使用 `./scripts/reinit-workspace.sh` 重新初始化工作区**：
   * 这个脚本会为每个模块添加必要的 `replace` 指令
   * 这些 replace 指令**应该保留在 go.mod 中**，它们是开发期的正确状态
   * `go.work` + replace 指令共同确保本地模块互相引用正确
4. **不要手动删除 go.mod 中的 replace 指令**（开发期）：
   * 这些 replace 防止 go mod tidy 尝试从远程拉取不存在的版本
   * 发布时 `release.sh` 会自动清理它们
   * 提交时可以安全地 commit 包含 replace 的 go.mod

### 外部依赖的升级

1. 要引入/升级外部库时，先用 `go get` 指定版本。
2. 然后 `go mod tidy`。
3. 把更新过的 `go.mod` / `go.sum` 提交，这没问题，因为外部依赖对所有人都是同一个远程 tag，不会有 workspace 覆盖的复杂性。

---

## 发布前（准备打 tag 的那一刻）

**使用自动化脚本**：`./scripts/release.sh v0.x.y`

该脚本会自动完成以下步骤（从底层到高层，逐层发布）：

1. **清理 replace 指令**：删除所有开发期遗留的本地 replace 指令
2. **更新依赖版本**：将内部依赖更新到刚发布的版本（`go mod edit -require=...`）
3. **运行 go mod tidy**：使用 `GOPROXY=direct` + 重试机制确保 go.sum 正确
4. **提交变更**：commit 所有 go.mod 和 go.sum 的修改
5. **打标签并推送**：`git tag <module>/<vX.Y.Z>` + `git push origin <module>/<vX.Y.Z>`
6. **重复上述步骤**：自动处理所有模块，按依赖顺序逐层发布

**手动发布**（如需手动控制）：
1. 从依赖最底层模块开始（core → runtimex → connectx → ...）
2. 删除所有 replace 指令：`go mod edit -dropreplace=go.eggybyte.com/egg/...`
3. 更新 require 版本：`go mod edit -require=go.eggybyte.com/egg/xxx@v0.x.y`
4. 运行 `GOPROXY=direct go mod tidy`
5. Commit 变更并打 tag

### 发布时如何正确 tidy？

**问题**：在 `release.sh` 中运行 `go mod tidy` 时，可能遇到 "unknown revision" 错误：

```
go: reading go.eggybyte.com/egg/core/go.mod at revision v0.3.0-alpha.2: unknown revision v0.3.0-alpha.2
```

**原因**：
1. Tag 刚推送到 GitHub，需要几秒钟时间让 Git 服务器完成处理。
2. 即使使用 `GOPROXY=direct`，某些传递依赖的测试包仍可能访问缓存。
3. GitHub 的 CDN 可能需要时间来传播新的 tag 信息。

**解决方案**：

```bash
# 使用 GOPROXY=direct + 重试机制
max_retries=3
retry_delay=5

for attempt in $(seq 1 $max_retries); do
    if [ $attempt -gt 1 ]; then
        echo "Retry $attempt/$max_retries (waiting ${retry_delay}s)..."
        sleep $retry_delay
    fi
    
    if GOPROXY=direct go mod tidy 2>&1; then
        echo "Success!"
        break
    fi
done
```

**为什么这样有效**：
- `GOPROXY=direct` 强制从 GitHub 直接拉取，绕过 proxy.golang.org
- 重试机制给 GitHub 时间来传播 tag（通常 5-10 秒足够）
- 失败后的警告不影响发布流程，用户 `go get` 时会自动修复

这样做的好处：
- ✅ go.sum 在发布时就是正确且完整的
- ✅ 下游用户拉取时不需要重新生成 go.sum
- ✅ 避免因 go.sum 不一致导致的 checksum mismatch 错误
- ✅ 符合 Go 模块的最佳实践

---

## 你真正要记住的 6 句话

* **`go.work` 解决"我本地要让多个模块一起跑"，不是解决"go.mod 的版本永远保持最新 HEAD"。**
* **内部模块之间的 go.mod 版本号只在发版前才更新，平时不用追着 HEAD 改。**
* **外部依赖的版本用 `go get` 控，再用 `go mod tidy` 清理。**
* **开发期保留 `replace` 指令在 go.mod 中，它们防止 tidy 拉远程假版本，可以安全提交。**
* **发布时 `release.sh` 会自动删除所有 replace 指令，确保发布的 go.mod 干净可复现。**
* **发布时要 tidy，使用 `GOPROXY=direct` + 重试机制避免 GitHub tag 传播延迟。**

照这个模型走，你就能：

* 自由同时改 core / runtimex / connectx，
* 下游项目还能稳定用你发出去的 tag，
* 而且不会在开发期被 `go mod tidy` 把每个模块都搞到引用奇怪的 pseudo-version。
