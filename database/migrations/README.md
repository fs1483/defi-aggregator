# 数据库迁移管理

## 📋 概述

本目录包含所有数据库迁移文件，采用标准的版本化迁移管理方式。

## 🗂️ 迁移文件命名规范

```
{version}_{description}.sql
```

**示例**:
- `001_initial_schema.sql` - 初始数据库架构
- `002_add_user_avatar.sql` - 添加用户头像字段
- `003_optimize_indexes.sql` - 优化索引

## 🔄 迁移执行顺序

迁移文件按数字顺序执行，确保数据库始终处于一致状态。

### **当前迁移列表**

| 版本 | 文件 | 描述 | 状态 |
|------|------|------|------|
| 001 | `001_initial_schema.sql` | 创建初始数据库架构 | ✅ 完成 |
| 002 | `002_future_migration.sql` | 未来的迁移 | 📋 计划中 |

## 🚀 迁移执行指南

### **手动执行迁移**

```bash
# 执行单个迁移
psql -h localhost -U admin -d defi_aggregator -f migrations/001_initial_schema.sql

# 执行所有迁移 (按顺序)
for file in migrations/*.sql; do
    echo "Executing migration: $file"
    psql -h localhost -U admin -d defi_aggregator -f "$file"
done
```

### **使用迁移工具 (推荐)**

#### **golang-migrate**
```bash
# 安装
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 执行迁移
migrate -path database/migrations -database "postgres://admin:password@localhost:5432/defi_aggregator?sslmode=disable" up

# 回滚迁移
migrate -path database/migrations -database "postgres://admin:password@localhost:5432/defi_aggregator?sslmode=disable" down 1
```

#### **Flyway (Java生态)**
```bash
# flyway.conf 配置
flyway.url=jdbc:postgresql://localhost:5432/defi_aggregator
flyway.user=admin
flyway.password=password
flyway.locations=filesystem:database/migrations

# 执行迁移
flyway migrate

# 查看状态
flyway info
```

## ✅ 迁移最佳实践

### **1. 事务安全**
```sql
-- 每个迁移文件都应该包含事务
BEGIN;

-- 迁移内容
CREATE TABLE ...;
ALTER TABLE ...;

-- 提交事务
COMMIT;
```

### **2. 回滚准备**
```sql
-- 为每个迁移准备对应的回滚脚本
-- down_001_initial_schema.sql
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS quote_responses;
-- ... 按依赖关系逆序删除
```

### **3. 数据完整性检查**
```sql
-- 迁移后验证
SELECT 'Migration completed successfully' as status;

-- 检查关键表
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM tokens;
```

## 🛡️ 安全注意事项

### **生产环境迁移检查清单**

- [ ] 备份数据库
- [ ] 在测试环境验证迁移
- [ ] 检查迁移执行时间
- [ ] 准备回滚方案
- [ ] 通知相关团队
- [ ] 监控迁移执行状态

### **迁移前备份**
```bash
# 创建备份
pg_dump -h localhost -U admin -d defi_aggregator > backup_before_migration_$(date +%Y%m%d_%H%M%S).sql
```

## 📊 迁移状态跟踪

### **创建迁移状态表**
```sql
-- 迁移版本跟踪表
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    dirty BOOLEAN NOT NULL DEFAULT FALSE,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### **记录迁移执行**
```sql
-- 在每个迁移文件末尾添加
INSERT INTO schema_migrations (version, dirty) 
VALUES ('001', FALSE) 
ON CONFLICT (version) DO UPDATE SET dirty = FALSE, executed_at = CURRENT_TIMESTAMP;
```

## 🔍 故障排除

### **常见问题**

#### **迁移失败回滚**
```sql
-- 如果迁移失败，手动回滚
ROLLBACK;

-- 检查数据库状态
SELECT * FROM schema_migrations;
```

#### **依赖关系错误**
```sql
-- 检查外键约束
SELECT 
    tc.table_name, 
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name 
FROM information_schema.table_constraints AS tc 
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY';
```

#### **索引冲突**
```sql
-- 检查现有索引
SELECT indexname, tablename, indexdef 
FROM pg_indexes 
WHERE tablename IN ('users', 'tokens', 'transactions');
```

---

**记住**: 迁移是不可逆的操作，务必在生产环境执行前充分测试！
