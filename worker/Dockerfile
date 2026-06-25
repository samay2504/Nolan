FROM rust:bookworm AS rust-builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    clang libclang-dev protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Cache dependencies by building a dummy project first
COPY worker/Cargo.tom[l] worker/Cargo.loc[k] ./
ENV CARGO_NET_RETRY=10 \
    CARGO_HTTP_TIMEOUT=120 \
    CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
RUN mkdir src && echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

# Build actual source
COPY worker/ ./
COPY proto/ /proto/
RUN touch src/main.rs && cargo build --release

FROM debian:bookworm-slim

# Install runtime dependencies including ffmpeg
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Copy Rust binary
COPY --from=rust-builder /app/target/release/nolan-worker /usr/local/bin/

RUN ldconfig && \
    useradd -m -u 1000 worker

USER worker

ENTRYPOINT ["nolan-worker"]
