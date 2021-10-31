FROM alpine AS builder

RUN apk update
RUN apk add go olm-dev g++
RUN go install git.hnitbjorg.xyz/~edwargix/msc-link-bot@latest

FROM alpine

COPY --from=builder /root/go/bin/msc-link-bot /usr/local/bin/msc-link-bot
RUN apk add olm && \
    adduser -h /msclinkbot -S msclinkbot

USER msclinkbot
VOLUME /msclinkbot
WORKDIR /msclinkbot

CMD /usr/local/bin/msc-link-bot
