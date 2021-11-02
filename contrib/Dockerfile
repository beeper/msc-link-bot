FROM alpine AS builder

RUN apk update
RUN apk add g++ go make olm-dev
COPY . .
RUN make

FROM alpine

COPY --from=builder /msc-link-bot /usr/local/bin/msc-link-bot
RUN apk add olm && \
    adduser -h /msclinkbot -S msclinkbot

USER msclinkbot
VOLUME /msclinkbot
WORKDIR /msclinkbot

CMD /usr/local/bin/msc-link-bot
