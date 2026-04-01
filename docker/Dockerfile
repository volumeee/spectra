# Stage 1: Build
FROM golang:1.23-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /bin/spectra ./cmd/spectra/
RUN CGO_ENABLED=0 go build -o /bin/spectra-cli ./cmd/spectra-cli/

RUN mkdir -p /bin/plugins && \
    for dir in plugins/*/; do \
        name=$(basename "$dir"); \
        CGO_ENABLED=0 go build -o "/bin/plugins/$name" "./$dir"; \
        if [ -f "$dir/plugin.json" ]; then \
            cp "$dir/plugin.json" "/bin/plugins/${name}.json"; \
        fi; \
    done

# Stage 2: Runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    fonts-liberation \
    fonts-noto-cjk \
    fonts-noto-color-emoji \
    dumb-init \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash spectra

COPY --from=builder /bin/spectra /usr/local/bin/spectra
COPY --from=builder /bin/spectra-cli /usr/local/bin/spectra-cli
COPY --from=builder /bin/plugins /home/spectra/plugins
COPY spectra.example.yaml /home/spectra/spectra.yaml

RUN chown -R spectra:spectra /home/spectra

USER spectra
WORKDIR /home/spectra

ENV SPECTRA_PLUGINS_DIR=/home/spectra/plugins
ENV SPECTRA_SERVER_PORT=3000

EXPOSE 3000

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s \
    CMD curl -f http://localhost:3000/health || exit 1

ENTRYPOINT ["dumb-init", "--"]
CMD ["spectra"]
