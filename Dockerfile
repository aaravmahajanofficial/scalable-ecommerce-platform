# Stage 1 - Build
FROM golang:1.24-bookworm AS build

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd/scalable-ecommerce-platform ./cmd/scalable-ecommerce-platform
COPY ./internal ./internal

# creates statically linked executable, it contains all the code it needs to run, strips debug info to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/scalable-ecommerce ./cmd/scalable-ecommerce-platform/


# Stage 2 - Minimal Runtime
FROM gcr.io/distroless/static-debian12

COPY --from=build /app/scalable-ecommerce /app/scalable-ecommerce

EXPOSE 8085

CMD [ "/app/scalable-ecommerce" ]