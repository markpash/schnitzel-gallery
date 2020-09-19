FROM rustlang/rust:nightly-alpine AS builder

WORKDIR /build
COPY . .
RUN apk add musl-dev && cargo build --release

FROM alpine:latest

WORKDIR /app
COPY ./static ./static
COPY ./notfound.jpg ./notfound.jpg
COPY --from=builder /build/target/release/schnitzel-gallery .

CMD ./schnitzel-gallery
