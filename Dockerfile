FROM golang:1.24.4 AS build

WORKDIR /app
COPY . .

RUN go mod download
RUN go vet -v
RUN make build

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /app/bin/asset-watcher /asset-watcher
CMD ["/asset-watcher"]
