FROM golang:1.24 as builder

# Install libpcap for packet capture
RUN apt-get update && apt-get install -y libpcap-dev

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o network-monitor ./cmd/monitor

# Final stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y libpcap0.8 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the executable from the builder stage
COPY --from=builder /app/network-monitor .
# Copy the example config
COPY config.yaml.example /app/config.yaml.example

# Give the binary the capability to capture packets
RUN apt-get update && apt-get install -y libcap2-bin && \
    setcap cap_net_raw,cap_net_admin=+ep /app/network-monitor && \
    apt-get remove -y libcap2-bin && apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# Expose the metrics port
EXPOSE 9090

# Run the application
CMD ["./network-monitor"] 