# 车辆入场请求完整链路说明

本文档以 **车辆入场 POST `/api/v1/parking/entry`** 为例，完整讲解一个 HTTP 请求从进入 Gin 框架到最终返回响应的完整执行路径。

## 一、总体架构分层

```
HTTP 请求 → Gin 路由 → 中间件链 → Handler → Service → Model(DB) → Handler 组装响应 → Gin 返回 JSON
```

| 层级 | 职责 | 代码位置 |
|------|------|---------|
| 路由层 | URL 与 Handler 绑定 | [main.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/main.go#L36-L74) |
| 中间件层 | 鉴权、CORS、角色校验 | [middleware/middleware.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/middleware/middleware.go) |
| Handler 层 | 参数解析、业务编排、响应组装 | [handlers/parking.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/handlers/parking.go) |
| Service 层 | 核心业务规则（计费算法等） | [services/parking_service.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/services/parking_service.go) |
| Model 层 | 数据结构定义 + GORM 数据库操作 | [models/models.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/models/models.go) |
| 工具层 | JWT Token 生成/解析 | [utils/jwt.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/utils/jwt.go) |
| 配置层 | YAML 配置加载 | [config/config.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/config/config.go) |

---

## 二、服务启动与路由注册（请求前）

### 2.1 依赖初始化

[main.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/main.go#L13-L36) 在服务启动时按顺序完成：

1. `config.LoadConfig("config.yaml")` → 加载 YAML 到 `*config.Config`
2. `models.InitDB(cfg)` → 建立 MySQL 连接 + GORM AutoMigrate 建表 + 插入默认管理员
3. `models.InitParkingSpaces(db, total)` → 初始化 100 个车位（首次启动）
4. 按「先 Service 后 Handler」顺序初始化对象：
   ```go
   parkingSvc := services.NewParkingService(cfg)          // 依赖 config
   parkingHandler := handlers.NewParkingHandler(db, parkingSvc) // 依赖 db + service
   ```

### 2.2 中间件与路由注册

[main.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/main.go#L28-L74) 中的路由定义：

```go
r := gin.Default()
r.Use(middleware.CORS())              // 全局中间件：所有请求先走 CORS

api := r.Group("/api/v1")
{
    api.POST("/auth/login", authHandler.Login)    // 公开接口，无鉴权
    api.POST("/auth/register", authHandler.Register)

    auth := api.Group("")
    auth.Use(middleware.JWTAuth(cfg.JWT.Secret))  // 分组中间件：下面所有接口先鉴权
    {
        auth.POST("/parking/entry", parkingHandler.Entry)   // ← 我们跟踪的接口
        auth.POST("/parking/exit", parkingHandler.Exit)
        ...
    }
}
```

**Gin 中间件执行顺序**是洋葱模型：请求按注册顺序穿过中间件，响应按反序返回。对于 `/parking/entry`，实际中间件链是：
```
CORS → JWTAuth → parkingHandler.Entry → JWTAuth（返回时）→ CORS（返回时）
```

---

## 三、请求执行完整步骤

### 步骤 0：客户端发起请求

```bash
curl -X POST http://localhost:8080/api/v1/parking/entry \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"plate_no": "京A12345"}'
```

---

### 步骤 1：CORS 中间件

[middleware.CORS()](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/middleware/middleware.go#L11-L22)

**职责**：设置跨域响应头，处理 OPTIONS 预检请求。

```go
c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
if c.Request.Method == "OPTIONS" {
    c.AbortWithStatus(http.StatusNoContent) // 预检直接返回，不进后续 handler
    return
}
c.Next() // 放行，进入下一个中间件
```

如果是普通 POST 请求，执行 `c.Next()` 进入 JWTAuth。

---

### 步骤 2：JWT 鉴权中间件

[middleware.JWTAuth(secret)](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/middleware/middleware.go#L24-L48)

**职责**：校验 Token 有效性，把用户信息注入 `gin.Context`。

执行流程：

1. 取 `Authorization` 请求头，校验格式为 `Bearer <token>`
2. 调用 [utils.ParseToken](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/utils/jwt.go#L32-L44) 解析 JWT：
   - 使用 HS256 算法 + 配置中的 `secret` 验签
   - 从 Payload 中取出 `user_id`、`username`、`role`
3. 解析成功则通过 `c.Set(key, value)` 写入 Context：
   ```go
   c.Set("user_id", claims.UserID)    // uint
   c.Set("username", claims.Username) // string
   c.Set("role", claims.Role)         // "admin" 或 "operator"
   c.Next() // 放行，进入业务 Handler
   ```
4. 任何一步失败都 `c.Abort()` 并返回 401，终止后续处理。

> 💡 `gin.Context` 是贯穿整个请求的上下文容器，Handler 层通过 `c.Get("username")` 就能拿到当前登录用户。

---

### 步骤 3：Handler 层 — Entry 方法

[handlers/parking.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/handlers/parking.go#L32-L83)

**职责**：参数校验 → 业务流程编排 → 调 DB → 组装响应。Handler 只做「编排」，具体计费算法交给 Service，SQL 细节交给 GORM。

#### 3.1 参数绑定与校验

```go
type EntryReq struct {
    PlateNo string `json:"plate_no" binding:"required"`
}
var req EntryReq
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
    return
}
```
- `ShouldBindJSON` 把请求 Body JSON 反序列化到结构体
- `binding:"required"` 由 Gin 的 validator 自动校验 `plate_no` 不能为空

#### 3.2 业务校验 1：车辆是否已在场

```go
var existing models.ParkingRecord
h.db.Where("plate_no = ? AND status = ?", req.PlateNo, "parking").First(&existing)
if existing.ID > 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "该车辆已在场内"})
    return
}
```
- GORM 的链式调用最终生成 SQL：`SELECT * FROM parking_records WHERE plate_no = '京A12345' AND status = 'parking' LIMIT 1`

#### 3.3 业务校验 2：是否有空闲车位

```go
var space models.ParkingSpace
h.db.Where("status = ?", "free").First(&space)
if space.ID == 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "没有空闲车位"})
    return
}
```
- 取 `status = 'free'` 的第一个车位（GORM 按主键升序）

#### 3.4 月卡车辆判断

```go
var monthlyCard models.MonthlyCard
isMonthly := false
now := time.Now()
h.db.Where("plate_no = ? AND status = ? AND start_date <= ? AND end_date >= ?",
    req.PlateNo, "active", now, now).First(&monthlyCard)
if monthlyCard.ID > 0 {
    isMonthly = true
}
```
- 查 `monthly_cards` 表，该车牌是否有「生效中」的月卡

#### 3.5 从 Context 取当前操作员

```go
operator, _ := c.Get("username") // 来自 JWTAuth 中间件 c.Set 的值
```

#### 3.6 创建停车记录（事务外的两次写）

```go
record := models.ParkingRecord{
    PlateNo:   req.PlateNo,
    SpaceNo:   space.SpaceNo,
    EntryTime: now,
    IsMonthly: isMonthly,
    Operator:  operator.(string),
    Status:    "parking",
}
h.db.Create(&record)   // INSERT 到 parking_records，返回后 record.ID 已被填充
```

```go
space.Status = "occupied"
space.PlateNo = req.PlateNo
space.RecordID = &record.ID // 关联刚创建的记录 ID
h.db.Save(&space)     // UPDATE parking_spaces SET status='occupied', ...
```

#### 3.7 返回响应

```go
c.JSON(http.StatusOK, gin.H{
    "message":    "进场成功",
    "record":     record,      // 序列化时会带上 record.ID
    "is_monthly": isMonthly,
})
```

> ⚠️ 注意：这里两次 DB 操作（Create + Save）没有包在事务里，是个潜在风险点（Create 成功后 Save 失败会导致数据不一致）。可后续用 `h.db.Transaction(func(tx *gorm.DB) error { ... })` 优化。

---

### 步骤 4：GORM 与 MySQL 交互

[models.ParkingRecord](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/models/models.go#L33-L51) 结构体通过 GORM tag 映射到表结构。

以 `h.db.Create(&record)` 为例，GORM 内部执行：

1. 根据结构体名 `ParkingRecord` 自动蛇形成 `parking_records` 表名
2. 根据字段 tag（`gorm:"primaryKey"`、`gorm:"size:20;not null"` 等）生成 INSERT SQL
3. 执行 SQL 后把数据库生成的自增主键 `ID` 写回 `record.ID`

实际执行的 SQL 大致为：
```sql
INSERT INTO parking_records 
  (plate_no, space_no, entry_time, is_monthly, operator, status, created_at, updated_at)
VALUES 
  ('京A12345', 'A001', '2024-01-15 10:00:00', 0, 'operator01', 'parking', NOW(), NOW());
```

---

### 步骤 5：响应返回 Gin 框架

`c.JSON(200, ...)` 会：
1. 把 `gin.H` 序列化为 JSON 字符串
2. 写入 `Content-Type: application/json; charset=utf-8` 响应头
3. 写入 HTTP 状态码 200
4. 把缓冲区内容刷回客户端

**中间件洋葱回程**：响应按 `Entry → JWTAuth → CORS` 的反序再走一遍（如果中间件里 `c.Next()` 后有逻辑的话会执行，本项目中间件没有后置逻辑）。

---

## 四、出场请求的额外链路（计费部分）

出场 `POST /parking/exit` 比入场多了一步 Service 调用：

```
Exit Handler → ParkingService.CalculateFee → 按自然日切分 + 区间重叠公式 → 返回 FeeBreakdown
   ↓
写入 day_duration / night_duration / day_fee / night_fee / fee 到 parking_records
   ↓
写 bills 表（金额 > 0 时）
   ↓
返回 breakdown 明细
```

核心代码在 [handlers/parking.go Exit 方法](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/handlers/parking.go#L85-L149)，调用 [services/parking_service.go CalculateFee](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo22/project22/services/parking_service.go#L25-L30)。

---

## 五、关键数据流动汇总

| 阶段 | 关键对象 | 流向 |
|------|---------|------|
| 鉴权 | `Claims{UserID, Username, Role}` | JWT Payload → `utils.ParseToken` → `c.Set()` → Handler 中 `c.Get()` |
| 业务 | `ParkingRecord{ID, PlateNo, ...}` | Handler 组装结构体 → `gorm.Create()` → DB 回填 ID → JSON 序列化返回 |
| 车位 | `ParkingSpace{ID, SpaceNo, Status, ...}` | DB 查询 free → 修改为 occupied → `gorm.Save()` 回写 |
| 计费 | `FeeBreakdown{...}` | `ParkingService.CalculateFee()` → 回填到 ParkingRecord 字段 + JSON 返回 |
| 配置 | `ParkingConfig{...}` | YAML 文件 → `config.LoadConfig` → Service 持有引用 → 计费时读取 |

---

## 六、Gin Context 全链路传递图解

```
POST /api/v1/parking/entry
      │
      ▼
┌───────────────┐
│  CORS 中间件  │  设置响应头
└───────┬───────┘
        │ c.Next()
        ▼
┌───────────────┐
│ JWTAuth 中间件│  ParseToken → c.Set("user_id", ...)
└───────┬───────┘
        │ c.Next()
        ▼
┌───────────────┐
│ Entry Handler │  c.Get("username") → 写 DB → c.JSON()
└───────┬───────┘
        │ 返回
        ▼
┌───────────────┐
│ JWTAuth 中间件│  无后置逻辑
└───────┬───────┘
        │
        ▼
┌───────────────┐
│  CORS 中间件  │  无后置逻辑
└───────┬───────┘
        ▼
  HTTP 200 OK
```
