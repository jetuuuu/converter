FROM ubuntu
MAINTAINER Nikita <galnikrom@gmail.com>

RUN apt-get update -y
RUN apt-get install ffmpeg -y

RUN apt-get install --no-install-recommends -y \
    ca-certificates \
    curl \
    mercurial \
    git-core
RUN curl -s https://storage.googleapis.com/golang/go1.10.linux-amd64.tar.gz | tar -v -C /usr/local -xz

ENV GOPATH /go
ENV GOROOT /usr/local/go
ENV PATH /usr/local/go/bin:/go/bin:/usr/local/bin:$PATH

RUN mkdir /audio
RUN mkdir -p /go/src/github.com/jetuuuu/converter
COPY . /go/src/github.com/jetuuuu/converter/
WORKDIR /go/src/github.com/jetuuuu/converter/
RUN go build -o main

EXPOSE 8080

ENTRYPOINT [ "/go/src/github.com/jetuuuu/converter/run.sh" ]
