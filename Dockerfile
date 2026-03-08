FROM alpine:3.19 AS build
RUN --mount=type=cache,target=/var/cache/apk apk add git
ADD --chmod=755 https://github.com/wow-look-at-my/go-toolchain/releases/latest/download/go-toolchain_linux_amd64 /usr/local/bin/go-toolchain
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-toolchain go-toolchain

FROM alpine:3.19
COPY --from=build /src/build/frontend /bin/frontend
ENTRYPOINT ["/bin/frontend"]
