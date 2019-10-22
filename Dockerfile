FROM scratch

COPY multipass /
COPY multipass.conf /etc/multipass/multipass.conf

ENTRYPOINT ["/multipass"]
