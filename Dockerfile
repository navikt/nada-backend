FROM golang:1.19-alpine as builder
RUN apk add --no-cache git make
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /src
COPY go.sum go.sum
COPY go.mod go.mod
RUN go mod download
COPY . .
RUN make test
RUN make linux-build

FROM alpine:3
WORKDIR /app
COPY --from=builder /src/nada-backend /app/nada-backend
CMD ["/app/nada-backend"]
