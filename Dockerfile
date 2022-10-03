# build stage
FROM golang:buster as backend-builder
ENV GO111MODULE=on
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build cmd/torq/torq.go

# frontend build stage
FROM node:buster-slim as frontend-builder
WORKDIR /app
COPY web/package*.json ./
RUN npm install --legacy-peer-deps
COPY web/. .
RUN TSX_COMPILE_ON_ERROR=true ESLINT_NO_DEV_ERRORS=true npm run build

# final stage
FROM debian:buster-slim
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend-builder /app/torq /app/
COPY --from=frontend-builder /app/build /app/web/build
RUN useradd -ms /bin/bash torq
RUN apt-get install bash
ENV GIN_MODE=release
WORKDIR /app
USER torq
ENTRYPOINT ["./torq"]
