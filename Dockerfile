FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-greenhouse"]
COPY baton-greenhouse /