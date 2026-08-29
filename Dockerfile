FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod .
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /salomat-courier-server .

FROM alpine:3.20
RUN adduser -D -H appuser
USER appuser
COPY --from=build /salomat-courier-server /salomat-courier-server
EXPOSE 8080
CMD ["/salomat-courier-server"]
