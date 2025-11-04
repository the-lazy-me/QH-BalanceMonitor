FROM golang:1.21 AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum* ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -o balance-monitor main.go

# 运行阶段
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/balance-monitor .

EXPOSE 8080

CMD ["./balance-monitor"]

