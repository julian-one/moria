FROM golang:1.26-alpine AS build

# Git lets the build stamp vcs.revision into build info for `moria --version`.
RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -o /moria .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /moria /usr/local/bin/moria
COPY --from=build /app/schema ./schema

EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/moria"]
CMD ["serve"]
