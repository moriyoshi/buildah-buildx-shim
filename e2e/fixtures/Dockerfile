# Tiny image whose output depends on a build arg, so the test can prove the
# build actually ran with the value plumbed through compose -> bake -> buildah.
FROM docker.io/library/busybox:latest
ARG GREETING=default
RUN echo "$GREETING" > /greeting.txt
CMD ["cat", "/greeting.txt"]
