FROM google/kythe-base

RUN apt-get update && apt-get install -y --no-install-recommends \
    asciidoc source-highlight highlight graphviz gcc g++ && apt-get clean

# crosstool and grte
RUN mkdir -p /tmp/build
ADD crosstoolv17-gcc-4.8.x-grtev3-runtimes_1.0-63149863_amd64.deb /tmp/build/
ADD grtev3-runtimes_1.0-62117965_amd64.deb /tmp/build/
ADD crosstoolv17-gcc-4.8.x-grtev3-devel_1.0-63149863_amd64.deb /tmp/build/
ADD grtev3-headers_1.0-62117965_amd64.deb /tmp/build/
RUN dpkg -i /tmp/build/grtev3-runtimes_1.0-62117965_amd64.deb
RUN dpkg -i /tmp/build/grtev3-headers_1.0-62117965_amd64.deb
RUN dpkg -i /tmp/build/crosstoolv17-gcc-4.8.x-grtev3-runtimes_1.0-63149863_amd64.deb
RUN dpkg -i /tmp/build/crosstoolv17-gcc-4.8.x-grtev3-devel_1.0-63149863_amd64.deb
RUN rm -rf /tmp/build
ENV LD_LIBRARY_PATH /usr/crosstool/v17/gcc-x86_64-grtev3-linux-gnu/x86_64-grtev3-linux-gnu/lib64/:$LD_LIBRARY_PATH

# Helper script to set language tooling flags
ADD campfire /usr/local/bin/
ENV CAMPFIRE_ROOT /repo

ENTRYPOINT ["campfire"]
