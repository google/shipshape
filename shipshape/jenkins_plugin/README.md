
# Build and Install

To build and then install the google analysis plugin follow the below steps.

1. Build the plugin.

 The plugin needs jar files from kythe and shipshape to build.
 You can build and update the plugin dependencies by running:

    `$ ./update-dependencies.sh`

 To build the plugin you run:

    `$ ./build_plugin.sh`

 This will generated a .hpi file under the target directory in the root
 plugin directory.

2. Install the plugin in Jenkins.

 Make sure to uninstall any current version of the plugin. An installed plugin
 will be listed under 'Manage Jenkins' > 'Manage Plugins' and the 'Installed'
 tab. You uninstall by clicking the 'Uninstall' button next to the plugin and
 then follow the instructions to restart Jenkins.

 To install the plugin, select the 'Advanced' tab and under 'Upload Plugin'
 select to upload a file. Select the .hpi file you generated in the previous
 step and press 'Upload'. You may need to restart Jenkins.


3. Invoke the plugin in you project build.

 To invoke the plugin add it as a build step in the configuration of a build of
 your project.

 That is, in your project, select 'Configure' and add a 'Post Build' step in
 the 'Build' section. Select Shipshape from the list; the default configuration
 values should work on Linux and OS X. Make sure to 'Save' before you leave the
 page.


4. Test the plugin

 Invoke a build for your project, for instance, by explicitly pressing
 'Build Now'. This should start a build, and if you look at the console output
 of that build, you should see output from Shipshape.


# Testing locally

Run the following command from this directory:

    `$ mvn clean hpi:run`

This will start up Jenkins with this plugin installed.

