FROM golang:1.26-alpine AS build

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o /moria .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /moria /moria

EXPOSE 8080

ENTRYPOINT ["/moria"]
CMD ["serve"]
