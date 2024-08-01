FROM busybox:1.36.1 as assets

RUN addgroup -g 1001 nada && \
    adduser -u 1001 -G nada \
            -h /home/nada -D nada && \
    mkdir -p /home/nada/.config && \
    chown -R nada:nada /home/nada

COPY /nada-backend /nada-backend
RUN chown nada:nada /nada-backend
RUN chmod +x /nada-backend

FROM --platform=linux/amd64 gcr.io/distroless/static-debian12

COPY --chown=nada:nada --from=assets /etc/passwd /etc/passwd
COPY --chown=nada:nada --from=assets /home/nada /home/nada
COPY --chown=nada:nada --from=assets /home/nada/.config /home/nada/.config
COPY --chown=nada:nada --from=assets /nada-backend /home/nada/nada-backend

WORKDIR /home/nada
CMD ["/home/nada/nada-backend", "--config", "/home/nada/config.yaml"]
