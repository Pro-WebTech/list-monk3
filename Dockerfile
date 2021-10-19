FROM alpine:latest AS deploy
RUN apk --no-cache add ca-certificates
WORKDIR /listmonk
COPY listmonk .
COPY config.toml.sample config.toml
COPY config-demo.toml .
COPY auth_model.conf .
CMD ["./listmonk"]
EXPOSE 9000
