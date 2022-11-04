FROM alpine

WORKDIR /app
COPY csi-secret-injector .
ENTRYPOINT [ "./csi-secret-injector" ]
EXPOSE 8443
