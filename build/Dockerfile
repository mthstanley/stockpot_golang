# Build the Go Binary.
FROM golang:1.15.4 as builder
ARG VCS_REF
ENV CGO_ENABLED 0

# Create a location in the container for the source code. Using the
# default GOPATH location.
RUN mkdir -p /service
WORKDIR /service

# Copy the module files first and then download the dependencies. If this
# doesn't change, we won't need to do this again in future builds.
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the source code into the container.
WORKDIR /service
COPY . .

# Build the application.
WORKDIR /service/cmd/stockpot
RUN go build

# Run the Go Binary in Alpine.
FROM alpine:3.12
ARG BUILD_DATE
ARG VCS_REF
COPY --from=builder /service/cmd/stockpot/stockpot /service/stockpot
COPY configs /service
WORKDIR /service
EXPOSE 8000
CMD ["./stockpot", "--config", "stockpot.yaml", "server"]

LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.title="stockpot" \
      org.opencontainers.image.authors="Matthew Stanley <stanley.t.matthew@gmail.com>" \
      org.opencontainers.image.source="https://github.com/mthstanley/stockpot/cmd/stockpot" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.vendor="Stockpot"
