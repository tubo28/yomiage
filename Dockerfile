FROM golang:1.16 as build
RUN apt-get update && apt-get install build-essential -y
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
RUN go mod download
ADD . /workspace
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -ldflags '-linkmode external -extldflags "-static"' -o main

FROM alpine:3.13
COPY --from=build /workspace/main .
CMD ["./main"]
