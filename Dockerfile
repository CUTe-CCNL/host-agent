# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

# 安裝編譯依賴
RUN apk add --no-cache git

# 複製 go mod 檔案
COPY go.mod go.sum ./
RUN go mod download

# 複製程式碼
COPY . .

# 編譯
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o host-agent .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 從 builder 複製執行檔
COPY --from=builder /app/host-agent .
COPY --from=builder /app/config.yaml .

# 建立必要目錄
RUN mkdir -p /var/log/host-agent

EXPOSE 9100

CMD ["./host-agent", "-config", "/root/config.yaml"]
