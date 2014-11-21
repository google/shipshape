#!/bin/sh

CP="cp -f --preserve"

$CP /home/build/buildtools/grte/v3/releases/grtev3_1.0-62117965/debs/grtev3-headers_1.0-62117965_amd64.deb .
$CP /home/build/buildtools/grte/v3/releases/grtev3_1.0-62117965/debs/grtev3-runtimes_1.0-62117965_amd64.deb .
$CP /home/build/buildtools/crosstool/v17/releases/crosstoolv17-gcc-4.8.x-grtev3_1.0-63149863/debs/crosstoolv17-gcc-4.8.x-grtev3-devel_1.0-63149863_amd64.deb .
$CP /home/build/buildtools/crosstool/v17/releases/crosstoolv17-gcc-4.8.x-grtev3_1.0-63149863/debs/crosstoolv17-gcc-4.8.x-grtev3-runtimes_1.0-63149863_amd64.deb .

sha1sum -c sha1sums

