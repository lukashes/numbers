FROM alpine:3.3

RUN apk add --update ca-certificates && \
    rm -rf /var/cache/apk/* /tmp/*
RUN update-ca-certificates

COPY .build/numbers /numbers

ENTRYPOINT ["/numbers"]