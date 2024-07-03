
FROM golang:1.22.5 
ADD main.go /
WORKDIR /
RUN go mod init alert_webhook_listener && go build -o main . 
EXPOSE 3010
ENTRYPOINT ["/main"]
