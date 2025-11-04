# 余额监控系统

一个基于 Go + Next.js + SQLite 构建的智能余额监控与告警系统。

## ✨ 功能特性

- 🔐 **安全认证**：JWT token 认证，管理员密码保护
- 🔄 **自动监控**：每小时自动检查上游供应商余额
- 📊 **可视化仪表盘**：实时展示余额状态和历史趋势
- 📧 **SMTP 告警**：余额低于阈值时自动发送邮件通知
- 🎨 **美观界面**：基于 ShadCN UI 的现代化设计
- 🔧 **灵活配置**：支持管理多个上游供应商
- 📈 **历史记录**：完整的余额检查和通知历史

## 🔒 首次使用

系统首次运行时需要进行初始化设置：

1. 启动服务后访问 `http://localhost:3000`
2. 系统会自动跳转到初始化页面
3. 设置管理员用户名和密码（密码至少6位）
4. 初始化完成后使用账户登录即可使用系统

详细认证功能说明请查看 [认证功能说明.md](./认证功能说明.md)

## 🏗️ 技术栈

### 后端
- **Go 1.21+**：高性能后端服务
- **Gin**：Web 框架
- **SQLite**：轻量级数据库
- **JWT**：Token 认证
- **bcrypt**：密码加密
- **Cron**：定时任务调度
- **Gomail**：邮件发送

### 前端
- **Next.js 14**：React 框架
- **TypeScript**：类型安全
- **ShadCN UI**：组件库
- **Tailwind CSS**：样式框架
- **Recharts**：数据可视化

## 📦 快速启动

> 💡 **生产环境部署**：如需部署到服务器，请查看 [部署说明.md](./部署说明.md)

### 本地开发 - 使用启动脚本（推荐）

在项目根目录执行：
```bash
./启动服务.sh
```

停止服务：
```bash
./停止服务.sh
```

### 方式二：手动启动

#### 后端部署

1. 进入后端目录：
```bash
cd backend
```

2. 安装依赖：
```bash
go mod download
```

3. 配置环境变量（可选）：
```bash
export SERVER_PORT=:8080
export DATABASE_PATH=./balance_monitor.db
```

4. 运行服务：
```bash
go run main.go
```

### 前端部署

1. 进入前端目录：
```bash
cd frontend
```

2. 安装依赖：
```bash
npm install
```

3. 配置环境变量：
```bash
cp .env.example .env.local
```

编辑 `.env.local` 文件：
```env
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

4. 开发模式运行：
```bash
npm run dev
```

5. 生产构建：
```bash
npm run build
npm start
```

## 🚀 使用说明

### 1. 添加供应商

访问系统后，点击「供应商」标签页，添加上游供应商信息：

- **名称**：供应商标识名称
- **上游地址**：API 服务器地址（如 `https://api.example.com`）
- **Authorization**：认证令牌
- **New-API-User**：用户 ID（通常为 `1`）
- **启用状态**：是否启用监控

### 2. 配置 SMTP 通知

访问「设置」标签页，配置邮件通知：

- **SMTP 服务器**：如 `smtp.gmail.com`
- **端口**：通常为 `587`（TLS）或 `465`（SSL）
- **用户名**：SMTP 账号
- **密码**：SMTP 密码
- **发件人邮箱**：发送告警邮件的地址
- **收件人邮箱**：接收告警的邮箱（多个用逗号分隔）
- **告警阈值**：余额低于此值时发送告警

配置完成后，可以点击「发送测试邮件」验证配置。

### 3. 查看余额

系统会每小时自动检查一次余额，也可以在「仪表盘」页面点击「立即检查」手动触发。

仪表盘会显示：
- 总余额统计
- 各供应商当前余额
- 余额变化趋势图
- 详细余额列表

### 4. 查看通知历史

访问「通知」标签页，可以查看所有发送过的告警记录。

## 📋 API 接口

### 供应商管理

- `GET /api/providers` - 获取供应商列表
- `POST /api/providers` - 创建供应商
- `PUT /api/providers/:id` - 更新供应商
- `DELETE /api/providers/:id` - 删除供应商

### 余额查询

- `GET /api/balance` - 获取当前余额
- `GET /api/balance/history` - 获取历史记录
- `POST /api/balance/check` - 手动触发检查

### SMTP 设置

- `GET /api/smtp` - 获取 SMTP 设置
- `PUT /api/smtp` - 更新 SMTP 设置
- `POST /api/smtp/test` - 测试 SMTP 配置

### 通知历史

- `GET /api/notifications` - 获取通知历史

## 🔒 数据库结构

系统使用 SQLite 数据库，包含以下表：

- `providers` - 供应商信息
- `balance_history` - 余额历史记录
- `smtp_settings` - SMTP 配置
- `notifications` - 通知历史

## ⚙️ 配置说明

### 后端配置

通过环境变量配置：

- `SERVER_PORT` - 服务器端口（默认 `:8080`）
- `DATABASE_PATH` - 数据库文件路径（默认 `./balance_monitor.db`）

### 前端配置

在 `.env.local` 文件中配置：

- `NEXT_PUBLIC_API_URL` - 后端 API 地址

## 🐳 Docker 部署

### 后端 Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o balance-monitor main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/balance-monitor .
EXPOSE 8080
CMD ["./balance-monitor"]
```

### 前端 Dockerfile

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
EXPOSE 3000
CMD ["npm", "start"]
```

### Docker Compose

生产环境推荐配置（前端反向代理架构）：

```yaml
version: '3.8'
services:
  backend:
    build: ./backend
    # 后端不暴露到公网，仅容器内网络访问
    expose:
      - "8080"
    volumes:
      - ./data:/app/data
    environment:
      - DATABASE_PATH=/app/data/balance_monitor.db
  
  frontend:
    build: ./frontend
    ports:
      - "9597:3000"  # 映射到公网端口 9597
    environment:
      - BACKEND_URL=http://backend:8080
    depends_on:
      - backend
```

**架构优势**：
- ✅ 后端不暴露到公网，仅通过容器内部网络访问
- ✅ 浏览器所有请求发往同一端口，无跨域问题
- ✅ 只需开放一个端口 9597
- ✅ Next.js 自动代理 API 请求到后端

详细部署说明请查看 [部署说明.md](./部署说明.md)

## 📝 注意事项

1. **定时任务**：系统每小时自动检查一次余额，首次启动后会在下一个整点执行
2. **告警频率**：每个供应商在 24 小时内最多发送一次告警邮件，避免频繁打扰
3. **数据库备份**：建议定期备份 SQLite 数据库文件
4. **SMTP 配置**：建议使用专用的 SMTP 服务以确保邮件送达率

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

## 📧 联系方式

如有问题或建议，请通过 Issue 联系我们。

