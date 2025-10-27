<!-- df938337-31eb-47fb-a3e8-c9072ef6da08 3a317385-62fe-412c-a4e6-8104b970b86b -->
# Production-Ready Examples Refactoring

## Overview

清理示例服务以符合生产环境标准：删除 user-service 中的模拟逻辑，重构 minimal-connect-service 的目录结构。

## Changes

### 1. user-service: 移除 Mock Repository

**文件**: `examples/user-service/cmd/server/main.go`

- 删除 `mockUserRepository` 类型及其所有方法（第 139-287 行）
- 修改 `registerServices` 函数逻辑：
  ```go
  func registerServices(app *servicex.App) error {
      // Ensure database is configured
      db := app.DB()
      if db == nil {
          return fmt.Errorf("database is required but not configured: please set DB_DSN environment variable")
      }
      
      app.Logger().Info("using database-backed repository")
      userRepo := repository.NewUserRepository(db)
      
      // ... rest of initialization
  }
  ```

- 更新包文档，移除关于 mock repository 和 fallback 的说明
- 更新 Usage 部分，明确数据库为必需项

**理由**: 生产环境示例不应包含内存实现的后备方案，应该明确要求必需的依赖项。

### 2. minimal-connect-service: 重构目录结构

**操作**:

1. 创建 `cmd/server/` 目录
2. 移动 `main.go` → `cmd/server/main.go`
3. 更新 import 路径（如果 gen/ 目录引用发生变化）
4. 更新 `go.mod` 中的模块路径（如果需要）

**预期结构**:

```
minimal-connect-service/
├── cmd/
│   └── server/
│       └── main.go
├── api/
├── gen/
├── bin/
├── go.mod
└── go.sum
```

**理由**: 符合 egg 仓库标准，`cmd/<service>/` 应该是入口点的标准位置。

### 3. 验证和文档一致性

- 确保所有 GoDoc 注释为英文
- 确保文件头注释完整且准确
- 验证代码符合 egg 编码规范
- 检查错误处理和日志记录的一致性

## Impact

- **Breaking Change**: user-service 现在需要数据库才能启动
- **Benefits**: 
  - 更清晰的依赖要求
  - 更符合生产环境实践
  - 更简洁的代码（减少约 150 行）
  - 更标准的目录结构

## Files Modified

1. `examples/user-service/cmd/server/main.go` - 删除 mock，强制数据库
2. `examples/minimal-connect-service/cmd/server/main.go` - 新建（移动自根目录）
3. 可能需要更新构建脚本或 Makefile（如果存在）

### To-dos

- [ ] Remove mockUserRepository from user-service/cmd/server/main.go
- [ ] Update registerServices to require database with clear error message
- [ ] Update package documentation in user-service main.go
- [ ] Move minimal-connect-service/main.go to cmd/server/main.go
- [ ] Verify all imports and paths work correctly in both services