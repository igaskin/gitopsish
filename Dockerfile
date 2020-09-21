FROM golang:1.14.4 as builder

COPY . .
RUN go build -o gitopsish

FROM scratch

COPY --from=builder gitopsish .
RUN chmod +x gitopsish

ENTRYPOINT [ "gitopsish" ]