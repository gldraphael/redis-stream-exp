# Stage 0: build
FROM golang:1.25.3-alpine AS build

RUN adduser -D -u 10001 scratchuser
RUN apk add --no-cache git ca-certificates

WORKDIR /build
COPY go.mod go.sum   ./
RUN  go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# Stage 1: final
FROM scratch

ENV CONFIG_PATH=/app/config.yaml
EXPOSE 1323

WORKDIR /app
COPY --from=build /build/app         .
COPY --from=build /build/config.yaml ./config.yaml

COPY --from=build /etc/passwd /etc/passwd
USER scratchuser

CMD ["./app"]
