#!/bin/bash

/usr/sbin/sshd
./go_dispatcher &> /shipshape-output/go_dispatcher.log &
java -jar javac_dispatcher.jar &> /shipshape-output/javac_dispatcher.log &
if [ -z "$START_SERVICE" ]
then
  echo 'Running shipping container in streaming mode' > /shipshape-output/shipping_container.log
  ./shipshape
else
  ./shipshape --start_service &> /shipshape-output/shipping_container.log
fi

