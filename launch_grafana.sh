#!/bin/bash

docker run \
    -p 3000:3000 \
    -v $(pwd)/prom/prometheus.yaml:/etc/prometheus/prometheus.yml \
    -v grafana-storage:/var/lib/grafana \
    grafana/grafana & 2&1>/dev/null
