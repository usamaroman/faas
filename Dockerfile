FROM golang:1.25-alpine AS builder

RUN apk --no-cache add git

ARG DIR
WORKDIR /app

COPY go.work go.work.sum ./

COPY pkg ./pkg
COPY control_plane ./control_plane
COPY notifier ./notifier
COPY invoicer ./invoicer
COPY meter_agent ./meter_agent
COPY meter ./meter
COPY price_service ./price_service

WORKDIR /app/${DIR}

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o /bin/bin cmd/main.go

FROM alpine AS runner

WORKDIR /app
COPY --from=builder /bin/bin .

CMD [ "/app/bin" ]
