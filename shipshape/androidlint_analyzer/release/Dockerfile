FROM debian:wheezy

# Make sure all package lists are up-to-date
RUN apt-get update && apt-get upgrade -y &&  apt-get install -y \
   -q  --no-install-recommends sudo openjdk-6-jre \
   openjdk-6-jdk wget && apt-get clean

# Set up AndroidLint
# Get the jdk, then android.
# Update the necessary packages one at a time so we can accept each license as needed
RUN wget http://dl.google.com/android/android-sdk_r23-linux.tgz
RUN tar xvf android-sdk_r23-linux.tgz
ENV PATH /android-sdk-linux/tools:$PATH
RUN echo "y" | android update sdk --no-ui --filter platform-tools
RUN echo "y" | android update sdk --no-ui --filter tools
ENV PATH /android-sdk-linux/platform-tools:$PATH

# Set up AndroidLintAnalzer
ADD android_lint_service /android_lint_service
ADD endpoint.sh /endpoint.sh
EXPOSE 10005
ENTRYPOINT ["/endpoint.sh"]
