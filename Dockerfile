FROM alpine

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

EXPOSE 8014

COPY bsalliance /

ENTRYPOINT ["/bsalliance"]

CMD ["--help"]
