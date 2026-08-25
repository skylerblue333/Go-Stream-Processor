FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /sky-stream-aggregate ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /sky-stream-aggregate /sky-stream-aggregate
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/sky-stream-aggregate"]
