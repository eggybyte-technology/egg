# Dart/Flutter 命名兼容性

## 概述

为了支持 Flutter 前端服务创建，egg CLI 工具已经更新以兼容 Dart 的包命名规范。

## Dart 包命名规则

Dart 要求包名必须遵循以下规则：

1. **只能使用小写字母、数字和下划线** (`a-z`, `0-9`, `_`)
2. **不能以数字开头**
3. **不能是 Dart 保留关键字**（如 `class`, `if`, `for` 等）

特别注意：**连字符（`-`）不被允许**。

## 实现方案

### 1. 自动转换

当创建前端服务时，CLI 会自动将连字符转换为下划线：

```bash
# 用户输入（支持连字符）
egg create frontend admin-portal --platforms web

# 自动转换为 Dart 兼容的包名
# 创建的 Flutter 项目名称: admin_portal
```

**注意**：工具会显示转换信息：
```
[INFO] Converting service name to Dart-compatible package name: admin-portal -> admin_portal
```

### 2. 直接使用下划线

用户也可以直接使用下划线命名（推荐）：

```bash
egg create frontend admin_portal --platforms web
```

## 代码更改说明

### 更新的文件

1. **`cli/internal/generators/generators.go`**
   - 添加 `dartifyServiceName()` 函数：将连字符转换为下划线
   - 添加 `isValidDartPackageName()` 函数：验证 Dart 包名的合法性
   - 更新 `FrontendGenerator.Create()` 方法：自动转换服务名
   - 更新 `isValidServiceName()` 函数：接受下划线字符

2. **`cli/internal/configschema/config.go`**
   - 更新 `isValidServiceName()` 函数：接受下划线和连字符
   - 更新 `isValidImageName()` 函数：使用与服务名相同的规则
   - 更新错误提示信息：包含下划线的说明

3. **`cli/internal/lint/lint.go`**
   - 更新 `isValidServiceName()` 函数：接受下划线和连字符
   - 更新 lint 规则的建议信息

4. **`scripts/test-cli.sh`**
   - 更新测试脚本使用 `admin_portal` 而不是 `admin-portal`
   - 添加关于 Dart 命名兼容性的注释

## Dart 保留关键字列表

以下关键字不能用作包名：

```
abstract, as, assert, async, await, break, case, catch,
class, const, continue, covariant, default, deferred, do,
dynamic, else, enum, export, extends, extension, external,
factory, false, final, finally, for, function, get, hide,
if, implements, import, in, interface, is, late, library,
mixin, new, null, on, operator, part, required, rethrow,
return, set, show, static, super, switch, sync, this,
throw, true, try, typedef, var, void, while, with, yield
```

## 最佳实践

### 推荐的命名方式

✅ **推荐**：直接使用下划线
```bash
egg create frontend user_portal --platforms web
egg create frontend admin_dashboard --platforms web
egg create frontend mobile_app --platforms android,ios
```

⚠️ **接受但会转换**：使用连字符（会自动转换）
```bash
egg create frontend user-portal --platforms web
# 自动转换为: user_portal
```

❌ **不推荐**：混合使用
```bash
egg create frontend user_portal-v2  # 避免这种混合方式
```

### 命名建议

1. **保持一致性**：在整个项目中使用相同的命名风格
2. **使用下划线**：直接使用下划线避免转换
3. **语义清晰**：使用描述性的名称，如 `admin_portal`, `user_dashboard`
4. **避免缩写**：使用完整的单词，除非缩写非常常见

## 验证

### 手动测试

创建一个测试项目验证：

```bash
# 初始化项目
egg init --project-name test --module-prefix github.com/test/project

# 创建前端服务（使用下划线）
egg create frontend test_app --platforms web

# 验证结构
ls -la frontend/test_app
ls -la frontend/test_app/lib

# 检查配置
egg check
```

### 自动测试

运行集成测试套件：

```bash
make test-cli
```

## 相关参考

- [Dart 包命名规范](https://dart.dev/tools/pub/pubspec#name)
- [Flutter Create 命令文档](https://docs.flutter.dev/reference/flutter-cli#flutter-create)
- [Egg CLI 文档](./egg-cli.md)

## 故障排除

### 问题：创建前端服务时报错 "Invalid service name"

**原因**：服务名包含非法字符（如大写字母、特殊符号等）

**解决方案**：
```bash
# 错误示例
egg create frontend AdminPortal  # ❌ 包含大写字母
egg create frontend admin@portal # ❌ 包含特殊字符

# 正确示例
egg create frontend admin_portal # ✅ 只使用小写字母和下划线
```

### 问题：egg check 报告 "Invalid service name"

**原因**：旧版本的 CLI 不支持下划线

**解决方案**：
```bash
# 重新构建 CLI
make build-cli

# 或者更新到最新版本
go install github.com/eggybyte-technology/egg/cli/cmd/egg@latest
```

## 版本历史

- **v0.0.1** (2025-01-20)
  - 初始实现 Dart 命名兼容性
  - 支持自动转换连字符为下划线
  - 添加 Dart 关键字验证

---

**注意**：本文档随 egg CLI 工具一起维护。如有问题或建议，请提交 Issue 或 Pull Request。

