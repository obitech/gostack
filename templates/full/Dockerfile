# Stage 1: Build frontend
FROM node:26-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
# Node 25+ no longer ships Corepack, so install the pnpm version pinned in
# package.json's packageManager field.
RUN npm install -g "$(node -p "require('./package.json').packageManager")" && \
    pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm run build

# Stage 2: Build Go binary
FROM golang:1.27.4-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /app-binary ./cmd/cli

# Stage 3: Runtime
FROM alpine:3.24
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app-binary .
EXPOSE 8080
CMD ["./app-binary", "run-server"]
