FROM golang:1.20-alpine as builder
RUN apk add --no-cache gcc musl-dev linux-headers git
WORKDIR /build
COPY . .
RUN go build -o prover-proxy ./cmd/prover/main.go ./cmd/prover/flag.go

FROM alpine:latest as runner
RUN apk add --no-cache ca-certificates
COPY --from=builder /build /usr/local/bin/
EXPOSE 6000
ENTRYPOINT ["prover-proxy"]
