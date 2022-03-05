FROM scratch
ENTRYPOINT ["/bin/kube-network-monitor"]
COPY kube-network-monitor /bin/kube-network-monitor
WORKDIR /workdir
