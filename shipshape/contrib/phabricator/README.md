# Phabricator Shipshape bridge
The code in this directory provides a bridge to report Shipshape results in
Phabricator as inline comments.

# Implementation
This code creates a nodejs webserver in a Docker container.
The webserver listens for very specific http requests, that come in the form of

/?phid=${target.phid}&rev=${buildable.revision}&diff=${buildable.diff}&secret={secret}

Once the server receives such a request, it will launch a script to clone/sync
 a git repo, it then uses "arc patch --diff" to apply the provided phabricator
 diff.
Once the diff is applied, the Shipshape CLI is run and the outputs of this run
 stored in a json file.

The server then processes the json file, matches it with files that are part of
 the code review, and posts inline comments in the phabricator code review.
Finally, it marks the phabricator build as completed.

On the Phabricator side, this bridge relies on Harbormaster and Herald.

# Configuration
## Phabricator side
Currently this implementation only supports one Repo per container,
if you have multiple Repos, configure multiple containers and multiple
phabricator builds.

In Phabricator's config enable prototype features.
Next, in Harbormaster, we create a build plan, that as its only step makes a
 HTTP Request, with as the URI the ip/dns of the server described above and the
 path listed above. Replace {secret} with a string of your choice.
Set "When complete" to "Wait for Message" to ensure shipshape processing
status is reported correctly in code reviews.

Create a Herald rule to automatically trigger shipshape "builds" for each code
 review:
When all of these conditions are met:
 Repository is any of <Your repo>
Take these actions every time this rule matches:
 Run build plans <the build plan you just created>

Note even though phabricators says any of, you should only list one repo here.

Finally, create a phabricator robot account for the shipshape build, note down
 the access token.


## Shipshape side
Have a machine/VM available that is accessible from the phabricator daemons.
Make sure to open the http port used (8080 by default) in the firewall.
Create the following set of directories:
-shipshape_root -> containing a .arcrc for the shipshape robot account,
   here you paste the access token from above.
-repos -> used to store repos so they don't have to be synced between docker
 container restarts
-output -> optionally to analyze output from shipshape runs

Run the following command to start the docker container:

docker run -t -i -p 8080:9090 -e SHIPSHAPE_HTTP_PORT=9090 \
  -e PHABRICATOR_SERVER=https:\/\/<address_of_phabricator_server> \
  -e SHIPSHAPE_SECRET=<the secret you chose> \
  -e GIT_REPO=<git repo address> \
  -e SHIPSHAPE_CATEGORIES="<shipshape categories to run>" \
  -v <shipshape_root>:/root \
  -v <repos>:/repos \
  -v <output>:/output \
  --privileged <location of docker container>/phipshape


