# Alpine Linux base container for LUSS

FROM alpine:latest
MAINTAINER Alexader Zaytsev "thebestzorro@yandex.ru"
ONBUILD RUN apk update && \
    apk upgrade && \
    apk add go ca-certificates

ADD luss /bin/luss
ADD templates /templates
RUN chmod 0755 /bin/luss && chown -R root:root /templates

EXPOSE 20080
VOLUME ["/data/conf/", "/data/luss/"]
WORKDIR /data
ENTRYPOINT ["/bin/luss"]
CMD ["-config", "/data/conf/luss.json"]
