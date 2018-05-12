FROM golang:latest
RUN mkdir /audio
RUN mkdir -p /go/src/github.com/jetuuuu/converter
COPY . /go/src/github.com/jetuuuu/converter/
WORKDIR /go/src/github.com/jetuuuu/converter/
RUN go build -o main

EXPOSE 8080

ENTRYPOINT [ "/go/src/github.com/jetuuuu/converter/run.sh" ]
