FROM grafana/k6:latest

WORKDIR /scripts

COPY index.js .

ENTRYPOINT ["k6"]
CMD ["run", "index.js"]
