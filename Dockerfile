FROM golang:1.24-alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# create statically linked executable, it contains all the code it needs to run
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/scalable-ecommerce-platform/

FROM alpine:latest

WORKDIR /app

RUN addgroup -S tempGroup && adduser -S tempUser -G tempGroup

COPY --from=build-stage /app/server /app/server

RUN chown tempUser:tempGroup /app/server && chmod +x /app/server

USER tempUser

EXPOSE 8085

ENTRYPOINT [ "/app/server" ]