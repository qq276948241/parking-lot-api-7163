# 停车场管理系统 API

基于 Go + Gin + GORM + MySQL 实现的 RESTful 停车场管理后端系统。

## 功能模块

- **车辆进出管理**：车牌号登记进场、出场自动计费、月卡车辆免费
- **车位管理**：总车位数、已占用/空闲查询、车位详情列表
- **月卡管理**：开卡、续费、查询、自动过期
- **收费账单**：按日/按月收入统计、CSV 明细导出

## 技术栈

- Go 1.21+
- Gin Web 框架
- GORM ORM
- MySQL 5.7+
- JWT 鉴权（HS256）

## 角色权限

| 角色 | 权限 |
|------|------|
| admin | 全部接口 |
| operator | 除用户管理、月卡增删改外的查询和操作接口 |

## 快速开始

### 1. 初始化数据库

```bash
mysql -u root -p < init.sql
```

或手动执行 init.sql 中的 SQL 语句。

### 2. 修改配置

编辑 `config.yaml`：

```yaml
database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: 你的密码
  dbname: parking_system
```

### 3. 安装依赖并运行

```bash
go mod tidy
go run main.go
```

服务默认启动在 `:8080`。

## 默认账号

| 用户名 | 密码 | 角色 |
|--------|------|------|
| admin | admin123 | 管理员 |
| operator01 | operator123 | 操作员 |

## API 接口

所有接口以 `/api/v1` 为前缀。除登录注册外，均需在 Header 中携带：

```
Authorization: Bearer <token>
```

### 鉴权

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | `/auth/login` | 登录 | 公开 |
| POST | `/auth/register` | 注册 | 公开 |
| GET | `/auth/me` | 当前用户信息 | 已登录 |
| POST | `/users` | 创建用户 | admin |
| GET | `/users` | 用户列表 | admin |
| DELETE | `/users/:id` | 删除用户 | admin |

### 车辆进出

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/parking/entry` | 车辆进场 `{ "plate_no": "京A12345" }` |
| POST | `/parking/exit` | 车辆出场 `{ "plate_no": "京A12345", "pay_type": "cash" }` |
| GET | `/parking/records` | 记录列表（支持 `plate_no`, `status`, `page`, `size`） |
| GET | `/parking/record/:id` | 单条记录详情 |

### 车位管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/spaces/status` | 车位占用概况（总/已占/空闲） |
| GET | `/spaces/list` | 车位列表（支持 `status` 过滤） |
| POST | `/spaces/add` | 批量添加车位（admin） `{ "count": 10 }` |

### 月卡管理

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/cards` | 开卡（admin） |
| POST | `/cards/renew` | 续费（admin） `{ "plate_no": "...", "months": 3 }` |
| GET | `/cards` | 月卡列表（支持 `plate_no`, `status`, `page`, `size`） |
| GET | `/cards/:id` | 月卡详情 |
| GET | `/cards/plate/:plate` | 按车牌号查有效月卡 |
| DELETE | `/cards/:id` | 删除月卡（admin） |

开卡请求体：
```json
{
  "plate_no": "京A12345",
  "owner_name": "张三",
  "phone": "13800138000",
  "months": 12
}
```

### 收费账单

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/bills/daily` | 日统计（参数 `date=2024-01-15`，默认今日） |
| GET | `/bills/monthly` | 月统计（参数 `month=2024-01`，默认本月） |
| GET | `/bills/list` | 账单列表（支持 `plate_no`, `start_date`, `end_date`, `page`, `size`） |
| GET | `/bills/export` | 导出 CSV（支持 `start_date`, `end_date`） |

## 计费规则

可在 `config.yaml` 中配置：

```yaml
parking:
  hourly_rate: 5        # 每小时5元
  max_daily_rate: 50    # 单日封顶50元
  free_minutes: 15      # 前15分钟免费
  monthly_card_price: 200  # 月卡单价/月
```

计费逻辑：
1. 停车时长 ≤ 免费分钟数：0 元
2. 超出部分按小时向上取整计费
3. 单日费用不超过封顶价
4. 月卡有效期内车辆全免费
