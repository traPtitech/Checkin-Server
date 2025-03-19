FROM --platform=$BUILDPLATFORM golang:1.24.1 AS build

WORKDIR /go/src/github.com/traPtitech/Checkin-Server

COPY ./go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

ENV GOCACHE=/tmp/go/cache

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/tmp/go/cache \
  go build -o /Checkin-Server 

FROM gcr.io/distroless/base:latest
WORKDIR /app
EXPOSE 3000

COPY --from=build /Checkin-Server ./

CMD ["./Checkin-Server"]