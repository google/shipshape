FROM google/cloud-dev-java:prod

# The default apt source list does not provide nodejs so we need to add a new source that does.
RUN echo "deb http://ftp.us.debian.org/debian wheezy-backports main" >> /etc/apt/sources.list
# Make sure instance is up to date
RUN apt-get update && apt-get upgrade -y  && \
   apt-get install -y  -qq --no-install-recommends \
# SUDO
  sudo \
# Packages needed for jshint
  nodejs-legacy curl moreutils \
# Packages needed for pylint
  pylint

# Setup jshint
RUN curl https://www.npmjs.org/install.sh | sponge | clean=no   sh
RUN npm install -g jshint

# Install Go, needed for the go vet analyzer
RUN curl -L -s http://golang.org/dl/go1.3.linux-amd64.tar.gz | tar -zx -C /usr/local
ENV PATH $PATH:/usr/local/go/bin
ENV GOPATH /go
ENV GOROOT /usr/local/go
