FROM centos
#USER root

MAINTAINER John McDowall <jmcdowall@paloaltonetworks.com>

RUN yum install -y epel-release
RUN yum install -y jq-devel.x86_64

RUN yum install -y net-tools make which rsync lshw docker-client openssh-clients libcurl.i686 iproute
RUN \
  mkdir -p /goroot && \
  curl https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz | tar xvzf - -C /goroot --strip-components=1
# Set environment variables.
ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH $GOROOT/bin:$GOPATH/bin:$PATH

# Define working directory.
WORKDIR /gopath/src/vnf-device-plugin
COPY vnf /usr/bin/vnf
COPY . .
RUN go build -o vnf-device-plugin
RUN cp vnf-device-plugin /usr/bin/vnf-device-plugin \
&& cp *.sh /usr/bin

ENTRYPOINT ["/usr/sbin/init"]
