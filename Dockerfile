FROM golang:1.24-alpine AS build
RUN apk add --no-cache curl && \
    curl -fsSL https://github.com/wow-look-at-my/go-toolchain/releases/latest/download/go-toolchain_linux_amd64 \
      -o /usr/local/bin/go-toolchain && \
    chmod +x /usr/local/bin/go-toolchain
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go-toolchain

FROM alpine:3.19
COPY --from=build /src/build/frontend /bin/frontend
ENTRYPOINT ["/bin/frontend"]
