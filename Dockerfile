FROM golang:1.27 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /moria .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /moria /moria
ENTRYPOINT ["/moria", "serve"]
