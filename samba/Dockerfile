FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        acl \
        attr \
        bind9-dnsutils \
        ca-certificates \
        dnsutils \
        krb5-user \
        procps \
        samba-ad-dc \
        smbclient \
        winbind \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY docker/entrypoint.sh /usr/local/bin/forlittle-samba-entrypoint
RUN chmod +x /usr/local/bin/forlittle-samba-entrypoint

VOLUME ["/etc/samba", "/var/lib/samba", "/var/cache/samba", "/var/log/samba"]

EXPOSE 53/tcp 53/udp 88/tcp 88/udp 135/tcp 137/udp 138/udp 139/tcp 389/tcp 389/udp 445/tcp 464/tcp 464/udp 636/tcp 3268/tcp 3269/tcp

ENTRYPOINT ["forlittle-samba-entrypoint"]
