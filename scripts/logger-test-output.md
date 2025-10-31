# Logger Functions Output Format

## 日志函数输出格式说明

### print_success
**格式**: `[✓] {message}`（整行绿色）
```bash
print_success "Operation completed successfully"
# 输出: [✓] Operation completed successfully (绿色)
```

### print_error
**格式**: `[✗] {message}`（整行红色）
```bash
print_error "Operation failed"
# 输出: [✗] Operation failed (红色)
```

### print_info
**格式**: `{message}`（整行白色，无前缀）
```bash
print_info "Initializing application..."
# 输出: Initializing application... (白色)
```

### print_warning
**格式**: `[!] {message}`（整行黄色）
```bash
print_warning "Configuration file not found"
# 输出: [!] Configuration file not found (黄色)
```

### print_debug
**格式**: `{message}`（整行洋红色，无前缀，仅在 DEBUG=true 时输出）
```bash
export DEBUG=true
print_debug "Debug mode enabled"
# 输出: Debug mode enabled (洋红色)
```

### print_section
**格式**: `• {section}`（白色圆点 + 白色加粗标题，朴实高级）
```bash
print_section "Configuration"
# 输出: • Configuration (白色圆点 + 白色加粗)
```

### print_header
**格式**: 带横线的标题（蓝色）
```bash
print_header "Test Header"
# 输出: 带横线的标题框
```

### print_step
**格式**: `{step}: {description}`（step 加粗）
```bash
print_step "Step 1" "Validating configuration"
# 输出: Step 1: Validating configuration
```

### print_command
**格式**: `[CMD] {command}`（亮青色加粗，整行高对比度）
```bash
print_command "make build"
# 输出: [CMD] make build (亮青色加粗，整行染色)
```

## 使用示例

运行测试脚本查看实际输出：
```bash
./scripts/test-logger.sh
```

