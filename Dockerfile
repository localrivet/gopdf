# Use an official Go runtime as a parent image (matching go.mod)
FROM golang:1.24-bookworm AS builder

# Set the working directory inside the container
WORKDIR /app

# Pre-copy go.mod and go.sum files to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project source code
COPY . .

# Build the runner executable
RUN CGO_ENABLED=0 GOOS=linux go build -o /gopdf-runner ./cmd/gopdf-runner

# Build the MCP server executable
RUN CGO_ENABLED=0 GOOS=linux go build -o /gopdf-mcp-server ./cmd/gopdf-mcp-server


# --- Final Stage ---
# Use a minimal base image like Debian Slim that includes necessary libraries for wkhtmltopdf
FROM debian:bookworm-slim

# Install wkhtmltopdf and dependencies required for headless execution
# Need ca-certificates for HTTPS URLs, fonts for rendering, xvfb for headless X server
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    wkhtmltopdf \
    xvfb \
    ca-certificates \
    fonts-liberation \
    && apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy the built Go binaries from the builder stage
COPY --from=builder /gopdf-runner /usr/local/bin/gopdf-runner
COPY --from=builder /gopdf-mcp-server /usr/local/bin/gopdf-mcp-server

# Set the entrypoint to run the MCP server within xvfb-run for headless operation
# xvfb-run handles the virtual X server needed by wkhtmltopdf
ENTRYPOINT ["xvfb-run", "/usr/local/bin/gopdf-mcp-server"]