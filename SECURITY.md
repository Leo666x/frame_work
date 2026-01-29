# 安全配置说明

## 环境配置安全

本项目已经移除了包含硬编码密码和API密钥的敏感文件，以确保代码安全。

### 被移除的敏感文件

- `power-ai-framework-v4/env/env.go` - 包含硬编码密码
- `__pycache__/` 目录 - Python缓存文件

### 安全配置步骤

1. **复制环境配置模板**
   ```bash
   cp power-ai-framework-v4/env/env.go.template power-ai-framework-v4/env/env.go
   ```

2. **创建环境变量文件**
   ```bash
   cp .env.example .env
   ```

3. **填入实际配置值**
   编辑 `.env` 文件，填入你的实际配置：
   - 数据库密码
   - API密钥
   - 服务器地址等

### 注意事项

- ⚠️ **永远不要**将 `.env` 文件提交到版本控制系统
- ⚠️ **永远不要**在代码中硬编码密码或API密钥
- ✅ 使用环境变量来管理敏感配置
- ✅ 定期更换密码和API密钥
- ✅ 使用强密码

### 原始安全隐患

原始代码中发现的安全问题：
- PostgreSQL 默认密码: `1qaz@WSX`
- MinIO 默认密码: `1qaz@WSX`  
- Weaviate API密钥: `WVF5YThaHlkYwhGUSmCRgsX3tD5ngdN8pkih`

这些默认值已被移除，现在通过环境变量安全管理。