# LUSS https://lus.su
FROM ubuntu:latest
MAINTAINER Alexader Zaytsev "thebestzorro@yandex.ru"

RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get -y install ca-certificates && \
    apt-get clean

ADD luss /bin/luss
ADD templates /templates
RUN chmod 0755 /bin/luss

EXPOSE 20080
VOLUME ["/data/conf/", "/data/luss/"]
WORKDIR /data
ENTRYPOINT ["/bin/luss"]
CMD ["-config", "/data/conf/luss.json"]