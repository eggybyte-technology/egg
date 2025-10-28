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
3. 不要在还没准备发版时到处跑 `go mod tidy`，尤其是高层模块（会污染它的 go.mod，试图去远程 resolve 你本地还没发 tag 的低层模块）。
4. 真的必须 tidy（例如你刚删除大量 import、go.mod 越来越脏）：

   * 允许你**临时**在当前模块的 go.mod 里加一条本地 replace：

     ```go
     replace github.com/eggybyte-technology/egg/core => ../core
     ```
   * 运行 `go mod tidy`
   * 运行成功后，把这条 replace 恢复/删除，不要 commit 上去。
   * 这是“开发期局部清扫”的安全逃生法。

### 外部依赖的升级

1. 要引入/升级外部库时，先用 `go get` 指定版本。
2. 然后 `go mod tidy`。
3. 把更新过的 `go.mod` / `go.sum` 提交，这没问题，因为外部依赖对所有人都是同一个远程 tag，不会有 workspace 覆盖的复杂性。

---

## 发布前（准备打 tag 的那一刻）

1. 从依赖最底层模块开始（core → runtimex → connectx → ...）。
2. 在该模块目录下**只更新 require 版本**（`go mod edit -require=...`）。
3. **不运行 `go mod tidy`**（原因见下文"为何发布时不 tidy"）。
4. Commit require 版本更新（`go.mod` 的修改）。
5. 打 tag：`git tag <module>/<vX.Y.Z>`，`git push origin <module>/<vX.Y.Z>`。
6. 然后到上层模块，把它的 `require` 更新到刚发的下层 tag，再重复步骤 2~5。
7. 完成整条链后，通知下游服务 `go get go.eggybyte.com/egg/xxx@vX.Y.Z`（下游的 `go get` 会自动处理 tidy）。

### 为何发布时不 tidy？

**问题**：在 `release.sh` 中，我们曾经尝试在推送 tag 后立即运行 `go mod tidy`，但遇到这样的错误：

```
go: reading go.eggybyte.com/egg/core/go.mod at revision v0.3.0-alpha.2: unknown revision v0.3.0-alpha.2
```

**原因**：
1. Tag 刚推送到 GitHub，Go 模块代理（proxy.golang.org）还没有索引到它。
2. 即使使用 `GOPROXY=direct`，某些传递依赖的测试包仍会尝试通过代理解析。
3. 导致 tidy 失败，并且可能部分修改 go.mod/go.sum 但未提交，造成 git 状态混乱。

**解决方案**：
- 发布时**只更新 require 版本，不 tidy**。
- go.mod 里的 require 版本是"我依赖这个模块的这个版本"。
- go.sum 在下游用户 `go get` 时自动生成（`go get` 会自动 tidy）。
- 本地开发时，通过 `go work` 解决依赖，不受影响。

这样做的好处：
- ✅ 避免 proxy 缓存延迟问题
- ✅ 简化发布流程
- ✅ 用户 `go get` 时会自动处理完整的依赖树和 go.sum
- ✅ 避免 git 状态混乱（部分修改未提交）

---

## 你真正要记住的 5 句话

* **`go.work` 解决"我本地要让多个模块一起跑"，不是解决"go.mod 的版本永远保持最新 HEAD"。**
* **内部模块之间的 go.mod 版本号只在发版前才更新，平时不用追着 HEAD 改。**
* **外部依赖的版本用 `go get` 控，再用 `go mod tidy` 清理。**
* **不要把临时的 `replace ../core` 之类的本地 hack 提交进主分支，它们只是为了让 `tidy` 不乱拉远程。**
* **发布时只更新 require 版本不 tidy，避免 proxy 缓存延迟导致的"unknown revision"错误。**

照这个模型走，你就能：

* 自由同时改 core / runtimex / connectx，
* 下游项目还能稳定用你发出去的 tag，
* 而且不会在开发期被 `go mod tidy` 把每个模块都搞到引用奇怪的 pseudo-version。
