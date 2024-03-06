##
## Build
##
FROM golang:1-bullseye AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o /app/server -pgo ./default.pgo ./cmd/rinha-server/main.go

##
## Deploy
##
FROM gcr.io/distroless/base-debian12
WORKDIR /app/
COPY --from=build /app/server /app/server
EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/app/server", "--port", "8080", "--host", "0.0.0.0"]
