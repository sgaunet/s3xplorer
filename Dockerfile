FROM alpine:3.21.3 AS alpine

FROM scratch AS final
# Copy the ca-certificates.crt from the alpine image
COPY --from=alpine /etc/ssl/certs/ /etc/ssl/certs/
COPY "resources" /
WORKDIR /
COPY s3xplorer /s3xplorer
USER s3xplorer
CMD ["/s3xplorer", "-f", "/cfg.yaml"]
