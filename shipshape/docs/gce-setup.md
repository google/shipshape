<!--
// Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
-->

<head>
  <!-- Ensure the Unicode → symbol renders correctly -->
  <meta charset="UTF-8">
</head>

# Setting up Shipshape on Google Compute Engine

We offer a prefabricated Shipshape image on Google Compute Engine. In order to
use it, all you need is a GCE project. You can create a project by following
[these instructions][1]. *Note that only the instructions in the section labeled
**`Set up a Google Cloud Platform project`** are necessary; don't follow the
instructions in the subsequent sections*.

## Creating an Instance with the GCE Console

The quickest way to get a running Shipshape instance on GCE is to use the
Developers Console to import our prefabricated image.

### Getting the Shipshape Image

To begin, navigate to the [Google Developers Console][2] and select
your project. Then, in the sidebar on the left, navigate to

    Compute → Compute Engine → Images

After this, click on the **`New Image`** button at the top of the page. This
should open the **`Create a new image`** form. Most of the fields presented to
you here are pretty self explanatory; however, for the final two you should use
the following values

|--------------------------|------------------------|
| **`Source`**             | `Cloud Storage file`   |
| **`Cloud Storage file`** | *`$image_bucket_path`* |

Make sure to take note of the value you provided in the **`Name`** field, and
then click **`Create`** at the bottom of the form.

### Creating an Instance from the Image

The next step is to create a VM instance from this image. From the sidebar on
the left, navigate to

    Compute → Compute Engine → VM instances

If this is the first instance you're adding to the project, click the **`Create
instance`** button presented to you. Otherwise click the **`New instance`**
button at the top-left of the interface.

You should now be faced with the **`Create a new instance`** form. For the most
part, you can configure the instance however you like. The only field that we
care about for now is labeled **`Boot disk`**; click the **`Change`** button to
bring up the **`Boot disk`** menu. Navigate to the tab labeled **`Your image`**
near the top of this menu. This should bring you to a list of GCE images
created by you. Choose the image that you created in the previous section, and
then click **`Select`** at the bottom of the menu; this should return you to the
**`Create a new instance`** form. Once you are satisfied with your VM's initial
configuration, click **`Create`** at the bottom of the form.

### Logging into the VM

Now you should have a running VM with shipshape installed. Once the machine is
up and running (this may take a few minutes) you can use `ssh` to obtain a
remote shell. There are two ways of accomplishing this:

  1. **Using GCE's builtin `SSH` tool**

    This is the easiest way to obtain a shell, as GCE will do most of the work
    for you. Like so many times before, use the sidebar on the left to navigate
    to

        Compute → Compute Engine → VM instances

    From the list of instances, click on the name of the VM you created in the
    previous section. This should bring you to a more detailed view of that VM.
    Click on the **`SSH`** button at the top, and an `ssh` session should open
    in a browser window.

  2. **Using your own `ssh` client**

    *TODO*

### Working with the VM

At this point you should have a remote shell on an Ubuntu 15.04 VM with
shipshape ready to go. There is just one step remaining before you are ready to
go -- you need to give yourself permission to use `docker`. Simply enter the
following command (without the `$`) into your shell

    $ sudo usermod -a -G docker ${USER?}

and then log out and back in.

### Congratulations!

You are ready to start writing a Shipshape analyzer!

## Creating an Instance with the `gcloud` tool

*TODO*
 
  [1]: https://cloud.google.com/compute/docs/linux-quickstart#set_up_a_google_cloud_platform_project
  [2]: https://console.developers.google.com
