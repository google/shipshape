
# Instance Setup

If this is your first time installing Shipshape, you will need to set up a dedicated instance.
If you already have one and just want to update to the latest version, skip to step 5.


1. Create a dedicated instance for Shipshape. This uses an image provided by Kubernetes and has docker
pre-installed.

	$ gcloud compute instances create <instance-name> \
	  --image container-vm-v20140826 \
	  --image-project google-containers \
	  --zone us-central1-a \
	  --machine-type f1-micro

2. Export port 10005 and 10007 from the GCE UI.
We have set up the default network settings for our project to open the appropriate ports


3. Log in to GCE instance and update everything.

 You can find the correct version of the below command under the SSH button on the overview page for the GCE instance you want to log into.

	$ gcloud compute --project <project-name> ssh --zone <zone> <instance-name>

 Update everything:
	$ sudo gcloud components update
	$ sudo apt-get update
	$ sudo apt-get upgrade

4. Change the docker settings.

 Docker communicates via a unix socket by default. In case you want to run the
 shipshape plugin on this instance, you should add an additional tcp socket.
 Add all sockets (the new tcp socket and the detault unix socket) as default
 options to docker by editing the file /etc/default/docker and replacing DOCKER_OPTS with:

 DOCKER_OPTS="-H tcp://127.0.0.1:4243 -H unix:///var/run/docker.sock"

 Restart docker to make it aware of the new tcp socket:

    $ sudo /etc/init.d/docker restart

 Add your user to the docker group to avoid having to run docker with sudo:

    $ sudo usermod -a -G docker "$USER"

 In case you cannot run 'docker ps' without sudo you need to log out and in for the usermod change to take effect.

5. Install and run the latest version of Shipshape.

 Get an access token to the docker registry (lasts for about an hour):

    $ sudo gcloud components update preview
    $ gcloud preview docker --server container.cloud.google.com --authorize_only

 Pull the image from the repository:

    $ docker pull container.cloud.google.com/_b_shipshape_registry/service:prod

 Run the image:

    $ sudo docker run -e START_SERVICE="true" -v /tmp:/shipshape-output -v /tmp:/shipshape-workspace -p 10005:10005 -p 10007:10007 -p 2222:22 --name=shipping_container -d container.cloud.google.com/_b_shipshape_registry/service:prod

 Verify that we have shipping-container running with:

    $ docker ps

6. Test the instance.

 On the GCE instance, verify that you can connect to the instance via SSH and ports used by Shipshape:

    $ curl -X POST -d '{"jsonrpc": "2.0", "id": 1, "method": "/ServerInfo/List"}' localhost:10005
    $ curl -X POST -d '{"jsonrpc": "2.0", "id": 1, "method": "/ServerInfo/List"}' localhost:10007

7. Run Shipshape on your files.

 TODO(ciera): end-to-end test not written yet
 From any GCE instance in your project, download the test client from GCS and run it with files in your repo.

    $ gsutil cp gs://analyzer-image-registry/shipping_container/test_shipshape test_shipshape --files=<FILES_IN_YOUR_REPO> --project_name=<YOUR_PROJECT_NAME>
