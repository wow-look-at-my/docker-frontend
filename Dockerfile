FROM scratch
COPY build/frontend /bin/frontend
ENTRYPOINT ["/bin/frontend"]
