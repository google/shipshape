FROM google/cloud-dev-nodejs:prod

RUN apt-get update && apt-get upgrade -y

RUN apt-get install -y php5 php5-curl
RUN mkdir /arc
RUN cd /arc && git clone https://github.com/phacility/libphutil.git
RUN cd /arc && git clone https://github.com/phacility/arcanist.git

ENV PATH /arc/arcanist/bin:$PATH
ADD async.js /shipshape/async.js
ADD conduit.js /shipshape/conduit.js
ADD server.js /shipshape/server.js
ADD analyze.sh /shipshape/analyze.sh
ADD shipshape /shipshape/cli/shipshape
ADD launch.sh /shipshape/launch.sh

RUN cd /shipshape/ && npm install winston

# Support Gerrit
ADD gitconfig /etc/gitconfig
ADD git-credential-gerrit.sh /google-cloud-sdk/bin/git-credential-gerrit.sh
RUN chmod +x /google-cloud-sdk/bin/git-credential-gerrit.sh

CMD [ "/shipshape/launch.sh"]
