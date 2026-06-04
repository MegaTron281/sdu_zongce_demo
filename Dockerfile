# 多阶段构建：前端 + 后端 → 精简运行时
# ─── 阶段 1：构建前端 ───
FROM node:22-alpine AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
RUN npx vite build

# ─── 阶段 2：编译后端 ───
FROM golang:1.23-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist
RUN go build -o /out/server ./cmd/server
RUN go build -o /out/migrate ./cmd/migrate
RUN go build -o /out/seed ./cmd/seed

# ─── 阶段 3：运行时 ───
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

# 应用文件放在 /app（不被平台挂载覆盖）
WORKDIR /app

COPY --from=backend-builder /out/server /app/server
COPY --from=backend-builder /out/migrate /app/migrate
COPY --from=backend-builder /out/seed /app/seed
COPY migrations/ /app/migrations/
COPY --from=backend-builder /src/frontend/dist /app/frontend/dist
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# 持久化数据目录 /data（平台会挂载，容器只往里写）
RUN mkdir -p /data/materials

ENV SERVER_ADDR=:8080
ENV UPLOAD_DIR=/data/materials
ENV GIN_MODE=release

EXPOSE 8080

ENTRYPOINT ["/app/docker-entrypoint.sh"]
