FROM alpine:3.19
COPY build/frontend /bin/frontend
ENTRYPOINT ["/bin/frontend"]
