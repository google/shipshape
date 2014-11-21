# This builds off of the production version of shipshape base.
# If you are testing changes that need a different version of
# base, you will need to (temporarily) modify this tag.
# There is no way to say "whatever tag I am using, use the same one."
FROM container.cloud.google.com/_b_shipshape_registry/base:prod

# Set up Shipshape
ADD javac_dispatcher_deploy.jar javac_dispatcher.jar
ADD go_dispatcher /go_dispatcher
ADD shipshape /shipshape
ADD endpoint.sh /endpoint.sh
EXPOSE 10005 10006 10007
ENTRYPOINT ["/endpoint.sh"]

