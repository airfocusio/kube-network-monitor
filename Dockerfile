FROM alpine:3.15
ENTRYPOINT ["/bin/kube-network-monitor"]
COPY kube-network-monitor /bin/kube-network-monitor
WORKDIR /workdir
