FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/orders-api .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -S app && adduser -S -G app app

COPY --from=build /out/orders-api /usr/local/bin/orders-api

USER app

EXPOSE 8080

CMD ["orders-api"]
