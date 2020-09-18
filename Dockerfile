FROM rustlang/rust:nightly AS builder

WORKDIR /build
COPY . .

RUN cargo build --release

CMD cargo run --release
