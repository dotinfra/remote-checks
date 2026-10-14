FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /remote-checks ./cmd/server

FROM scratch
COPY --from=build /remote-checks /remote-checks
COPY favicon.ico /favicon.ico
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/remote-checks"]
