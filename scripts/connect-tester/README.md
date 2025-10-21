# Connect Service Tester

Connect 服务测试工具，用于验证 Connect 端口的正确性和功能。

## 功能特性

- **全面测试**: 测试所有 Connect 服务端点
- **详细报告**: 提供详细的测试结果和错误信息
- **JSON 输出**: 支持 JSON 格式输出，便于程序化处理
- **超时控制**: 可配置的测试超时时间
- **多服务支持**: 支持测试多个不同的 Connect 服务

## 支持的服务

### Minimal Service
- `SayHello`: 测试基本问候功能
- `SayHelloStream`: 测试流式问候功能

### User Service
- `CreateUser`: 测试用户创建功能
- `GetUser`: 测试用户查询功能
- `ListUsers`: 测试用户列表功能

## 使用方法

### 基本用法

```bash
# 测试 minimal service
go run main.go http://localhost:8080 minimal-service

# 测试 user service
go run main.go http://localhost:8082 user-service
```

### 构建和使用

```bash
# 构建测试工具
go build -o connect-tester main.go

# 运行测试
./connect-tester http://localhost:8080 minimal-service
./connect-tester http://localhost:8082 user-service
```

### 在测试脚本中使用

```bash
# 运行完整的示例服务测试
./scripts/test.sh examples
```

## 输出格式

### 控制台输出

```
=== Connect Service Test Results ===
Service: minimal-service
Base URL: http://localhost:8080
Total Tests: 2
Passed: 2
Failed: 0
Duration: 1.234s

--- Test Details ---
✓ PASS SayHello (123ms)
✓ PASS SayHelloStream (456ms)
```

### JSON 输出

```json
{
  "service_name": "minimal-service",
  "base_url": "http://localhost:8080",
  "results": [
    {
      "service": "minimal-service",
      "test": "SayHello",
      "success": true,
      "duration": "123ms",
      "response": "Hello, TestUser!",
      "timestamp": "2024-01-01T12:00:00Z"
    }
  ],
  "start_time": "2024-01-01T12:00:00Z",
  "end_time": "2024-01-01T12:00:01Z",
  "total_tests": 2,
  "passed_tests": 2,
  "failed_tests": 0
}
```

## 错误处理

- **连接错误**: 服务不可达或网络问题
- **超时错误**: 请求超时
- **协议错误**: Connect 协议错误
- **业务错误**: 服务端业务逻辑错误

## 配置

### 超时设置

默认超时时间：
- 普通请求: 10 秒
- 流式请求: 15 秒

### 测试参数

- **Minimal Service**:
  - SayHello: 测试用户 "TestUser"，语言 "en"
  - SayHelloStream: 测试用户 "TestUser"，消息数量 3

- **User Service**:
  - CreateUser: 测试用户 "test@example.com", "Test User"
  - GetUser: 测试用户 ID "test-user-id"
  - ListUsers: 测试分页参数 page=1, page_size=10

## 集成到 CI/CD

测试工具返回适当的退出代码：
- `0`: 所有测试通过
- `1`: 有测试失败

可以在 CI/CD 管道中使用：

```bash
if ./connect-tester http://localhost:8080 minimal-service; then
    echo "Connect tests passed"
else
    echo "Connect tests failed"
    exit 1
fi
```

## 故障排除

### 常见问题

1. **服务不可达**
   - 检查服务是否正在运行
   - 验证端口是否正确
   - 检查防火墙设置

2. **构建失败**
   - 确保 Go 模块依赖正确
   - 检查网络连接
   - 验证 Go 版本兼容性

3. **测试超时**
   - 增加超时时间
   - 检查服务性能
   - 验证网络延迟

### 调试模式

启用详细日志输出：

```bash
# 查看详细错误信息
./connect-tester http://localhost:8080 minimal-service 2>&1 | tee test.log
```

## 开发

### 添加新服务测试

1. 在 `main.go` 中添加新的测试函数
2. 在 `main()` 函数中添加服务类型判断
3. 更新文档和示例

### 扩展测试功能

- 添加性能测试
- 支持自定义测试参数
- 添加负载测试功能
- 支持测试报告生成

## 许可证

本项目遵循 EggyByte Technology 的许可证条款。
