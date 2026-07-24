FROM dhi.io/golang:1.26-debian13-dev AS builder
WORKDIR /build
COPY . /build
RUN make clean
RUN --mount=type=cache,sharing=locked,target=/go/pkg/mod \
    --mount=type=cache,sharing=locked,target=/root/.cache/go-build \
    make build


FROM dhi.io/debian-base:trixie-dev AS provider
RUN --mount=type=cache,sharing=locked,target=/var/lib/apt/lists \
    --mount=type=cache,sharing=locked,target=/var/cache/apt \
    ls -FlAh /etc/apt/apt.conf.d/ \
    && rm -f /etc/apt/apt.conf.d/docker-clean \
    && apt-get --allow-releaseinfo-change update \
    && apt-get --yes --no-install-recommends install \
        fonts-noto \
        fonts-noto-cjk \
        fonts-noto-cjk-extra \
        fonts-noto-color-emoji \
        fonts-noto-extra \
        fonts-noto-mono \
        fonts-noto-ui-core \
        fonts-noto-ui-extra \
        fonts-noto-unhinted


#FROM dhi.io/alpine-base:3.23 AS runner  # NOPE: sqlite-go requires CGO_ENABLED
FROM dhi.io/debian-base:trixie AS runner
WORKDIR /bot

# We tried a bunch of things to convert SVG to PNG
# - inkscape does not cut it
# - chromium works, but it's huge
# - resvg works great
RUN mkdir -p ./lib/
COPY ./lib/resvg ./lib/

# resvg will need the Noto fonts since they are the ones we decided to use
COPY --from=provider /etc/fonts/conf.d/70-fonts-noto-cjk.conf /etc/fonts/conf.d
COPY --from=provider /etc/fonts/conf.avail/30-droid-noto.conf /etc/fonts/conf.avail
COPY --from=provider /etc/fonts/conf.avail/30-droid-noto-mono.conf /etc/fonts/conf.avail
COPY --from=provider /usr/share/fonts/opentype/noto/ /usr/share/fonts/opentype/noto/
COPY --from=provider /usr/share/fonts/truetype/noto/ /usr/share/fonts/truetype/noto/

# And now we copy the bot itself and its config
COPY --from=builder /build/mjbot ./
COPY --from=builder /build/.env ./
COPY --from=builder /build/.env.local ./
CMD ["/bot/mjbot"]
