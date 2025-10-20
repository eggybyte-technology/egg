# 发布指南 / Release Guide

本文档描述如何使用 GoReleaser 发布 Egg Framework 的新版本。

## 📋 前置条件

1. **安装 GoReleaser**
   ```bash
   make tools
   # 或者手动安装
   go install github.com/goreleaser/goreleaser/v2@latest
   ```

2. **GitHub Token**
   
   创建一个具有 `repo` 权限的 GitHub Personal Access Token：
   - 访问：https://github.com/settings/tokens/new
   - 勾选 `repo` 范围
   - 生成 token 并保存
   
   设置环境变量：
   ```bash
   export GITHUB_TOKEN=your_token_here
   ```

3. **Git 仓库配置**
   
   确保你的 git 配置正确：
   ```bash
   git config user.name "Your Name"
   git config user.email "your.email@example.com"
   ```

## 🚀 发布流程

### 1. 准备发布

#### 1.1 更新 CHANGELOG.md

在 `CHANGELOG.md` 中添加新版本的更新内容：

```markdown
## [v0.0.1] - 2024-01-20

### ✨ New Features
- Initial release of Egg Framework
- CLI tool for project scaffolding
- Core modules: log, errors, identity, utils
- Infrastructure modules: runtimex, connectx, configx, obsx, k8sx, storex

### 📚 Documentation
- Complete framework documentation
- CLI usage guide
- Quick start examples
```

#### 1.2 提交所有更改

```bash
# 添加所有文件到 git
git add .

# 提交更改
git commit -m "chore: prepare for v0.0.1 release"

# 推送到远程仓库
git push origin main
```

### 2. 本地测试发布

在实际发布之前，先在本地测试：

```bash
# 测试 GoReleaser 配置
make release-test

# 构建快照版本（不会创建 tag 或发布）
make release-snapshot
```

这会在 `dist/` 目录下生成测试构建：
```
dist/
├── egg_linux_amd64/
├── egg_darwin_amd64/
├── egg_darwin_arm64/
├── egg_windows_amd64/
└── checksums.txt
```

测试生成的二进制文件：
```bash
# Linux/macOS
./dist/egg_linux_amd64/egg version
./dist/egg_darwin_amd64/egg version
./dist/egg_darwin_arm64/egg version

# Windows
./dist/egg_windows_amd64/egg.exe version
```

### 3. 创建发布标签

#### 方式 1: 使用 Makefile（推荐）

```bash
make tag
# 输入版本号: v0.0.1
```

这会创建一个带注释的标签。

#### 方式 2: 手动创建

```bash
# 创建带注释的标签
git tag -a v0.0.1 -m "Release v0.0.1"
```

### 4. 推送标签

```bash
# 推送单个标签
git push origin v0.0.1

# 或推送所有标签
git push origin --tags
```

### 5. 发布到 GitHub

```bash
# 确保 GITHUB_TOKEN 已设置
export GITHUB_TOKEN=your_token_here

# 执行发布
make release-publish
```

GoReleaser 会：
1. ✅ 编译多平台二进制文件
2. ✅ 创建压缩包和校验和
3. ✅ 生成 CHANGELOG
4. ✅ 创建 GitHub Release
5. ✅ 上传所有资产

### 6. 验证发布

访问 GitHub Releases 页面验证：
```
https://github.com/eggybyte-technology/egg/releases/tag/v0.0.1
```

检查：
- ✅ Release 标题和描述
- ✅ 所有平台的二进制文件
- ✅ 压缩包和校验和文件
- ✅ 自动生成的 CHANGELOG

## 📦 发布产物

每个版本会生成以下文件：

### 二进制压缩包
- `egg_0.0.1_linux_amd64.tar.gz`
- `egg_0.0.1_darwin_amd64.tar.gz`
- `egg_0.0.1_darwin_arm64.tar.gz`
- `egg_0.0.1_windows_amd64.zip`

### 校验和
- `checksums.txt` - SHA256 校验和

### 文档
压缩包中包含：
- `LICENSE`
- `README.md`
- `CHANGELOG.md`
- `docs/*`

## 🔧 高级用法

### 预发布版本

创建预发布版本（alpha, beta, rc）：

```bash
# 创建预发布标签
git tag -a v0.0.1-beta.1 -m "Release v0.0.1-beta.1"
git push origin v0.0.1-beta.1

# 发布
make release-publish
```

GoReleaser 会自动检测并标记为 pre-release。

### 草稿发布

在 `.goreleaser.yaml` 中设置：
```yaml
release:
  draft: true
```

这样发布会创建为草稿，可以在发布前进行审查。

### 只构建特定平台

```bash
# 只构建 Linux amd64
goreleaser build --single-target

# 只构建 macOS
goreleaser build --config .goreleaser.yaml \
  --clean \
  --skip=validate \
  --rm-dist \
  --id egg-cli \
  --goos darwin
```

## 🎯 Go 模块版本

发布版本后，用户可以通过以下方式使用：

```bash
# 安装 CLI 工具
go install github.com/eggybyte-technology/egg/cli/cmd@v0.0.1

# 使用框架模块
go get github.com/eggybyte-technology/egg/core@v0.0.1
go get github.com/eggybyte-technology/egg/runtimex@v0.0.1
go get github.com/eggybyte-technology/egg/connectx@v0.0.1
go get github.com/eggybyte-technology/egg/configx@v0.0.1
go get github.com/eggybyte-technology/egg/obsx@v0.0.1
go get github.com/eggybyte-technology/egg/k8sx@v0.0.1
go get github.com/eggybyte-technology/egg/storex@v0.0.1
```

## 📝 版本规范

遵循 [Semantic Versioning 2.0.0](https://semver.org/)：

- **MAJOR** (v1.0.0): 不兼容的 API 变更
- **MINOR** (v0.1.0): 向后兼容的功能新增
- **PATCH** (v0.0.1): 向后兼容的问题修复

### 版本示例

- `v0.0.1` - 初始版本
- `v0.1.0` - 新增功能
- `v0.1.1` - Bug 修复
- `v1.0.0` - 首个稳定版本
- `v1.0.0-beta.1` - Beta 测试版本
- `v1.0.0-rc.1` - Release Candidate

## 🔄 快速发布检查清单

在发布前检查：

- [ ] 所有测试通过 (`make test`)
- [ ] Linter 无错误 (`make lint`)
- [ ] 更新了 `CHANGELOG.md`
- [ ] 更新了文档（如果有 API 变更）
- [ ] 提交并推送了所有更改
- [ ] 在本地测试了快照构建 (`make release-snapshot`)
- [ ] 设置了 `GITHUB_TOKEN` 环境变量
- [ ] 创建并推送了版本标签
- [ ] 执行发布命令 (`make release-publish`)
- [ ] 验证 GitHub Release 页面

## 🐛 常见问题

### 1. GoReleaser 找不到

**问题**: `goreleaser: command not found`

**解决**:
```bash
make tools
# 或
go install github.com/goreleaser/goreleaser/v2@latest
```

### 2. GITHUB_TOKEN 未设置

**问题**: `Error: GITHUB_TOKEN environment variable is not set`

**解决**:
```bash
export GITHUB_TOKEN=your_token_here
```

### 3. 标签已存在

**问题**: `tag already exists`

**解决**:
```bash
# 删除本地标签
git tag -d v0.0.1

# 删除远程标签（谨慎）
git push origin :refs/tags/v0.0.1
```

### 4. 发布失败后重试

如果发布过程中断：

```bash
# 删除失败的 release（在 GitHub 网页上）
# 删除对应的标签
git tag -d v0.0.1
git push origin :refs/tags/v0.0.1

# 重新创建标签并发布
git tag -a v0.0.1 -m "Release v0.0.1"
git push origin v0.0.1
make release-publish
```

## 📚 相关资源

- [GoReleaser 官方文档](https://goreleaser.com/)
- [GitHub Releases 文档](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)

## 🎉 完成

恭喜！你已经成功发布了 Egg Framework 的新版本。

---

**注意**: 首次发布需要确保 GitHub 仓库已经创建并且有适当的权限。

