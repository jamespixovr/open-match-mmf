# Gather dependencies and build the executable
FROM golang:1.19.4 as builder
ENV GO111MODULE=on

WORKDIR /app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service cmd/main.go


# Create the final image that will run the allocator service
FROM alpine:3.17

COPY --from=builder /app/service /app/service

RUN chmod o+x /app/service

CMD ["/app/service"]
