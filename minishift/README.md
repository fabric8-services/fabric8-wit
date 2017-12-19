## Start running dependent services on OpenShift

These instructions will help you run your services on OpenShift using MiniShift.

### Prerequisites


[Kedge](http://kedgeproject.org)

[MiniShift](https://docs.openshift.org/latest/minishift/getting-started/installing.html)

[Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)


### Installation (Linux)

##### Install Kedge

Following steps will download and install Kedge on your machine and put it in your $PATH. For more detailed information please visit kedgeproject.org

```
curl -L https://github.com/kedgeproject/kedge/releases/download/v0.5.1/kedge-linux-amd64 -o kedge
chmod +x ./kedge
sudo mv ./kedge /usr/local/bin/kedge
```

Verify installation by running following command, you should get version number.

```
kedge version
```

##### Install Minishift

Make sure you have all prerequisites installed. Please check the list [here](https://docs.openshift.org/latest/minishift/getting-started/installing.html#install-prerequisites)

Download and put `minishift` in your $PATH by following steps [here](https://docs.openshift.org/latest/minishift/getting-started/installing.html#manually)

It is very easy to put minishift in your PATH
```
<cd to downloaded_directory>
chmod +x ./minishift
sudo mv ./minishift /usr/local/bin/minishift
```

Verify installation by running following command, you should get version number.
```
minishift version
```


##### Install Kubectl

Please install and set up Kubectl on your machine by visiting [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Verify installation by running following command, you should get version number.
```
kubectl version
```

End with an example of getting some data out of the system or using it for a little demo

## Usage

Clone fabric8-wit repository
```
git clone git@github.com:fabric8-services/fabric8-wit.git
```

When you want to run fabric8-wit, fabric8-auth and databases on OpenShift use following command
```
cd minishift/
make dev-openshift
```

Please enter password when prompted, it is needed in order to make an entry in the `/etc/hosts`.
`minishift ip` gives the IP address on which MiniShift is running. This automation creates a host entry as `minishift.local` for that IP. This domain is whitelisted on fabric8-auth.

This build uses developer account for creating a project called `planner-services`.

Above command then automates the process of running the containers on OpenShift in MiniShift by using Kedge.

Now, when you want to use specific tag for WIT image or for AUTH image, use following `optional` values

```
WIT_IMAGE_TAG=<<WIT_TAG>> AUTH_IMAGE_TAG=<<AUTH_TAG>> make dev-openshift
```
When not specified any tag it is `latest` always.


### Run fabric8-ui locally
```
git clone git@github.com:fabric8-ui/fabric8-ui.git
npm install
FABRIC8_WIT_API_URL="http://`minishift ip`:30000/api/" FABRIC8_AUTH_API_URL="http://minishift.local:31000/api/" FABRIC8_REALM="fabric8-test" npm start
```
Once server is running, you can visit `http://localhost:3000` and login using prod-preview credentials.

## Cleanup
When you want to stop all the services running in MiniShift, use following command
```
make clean-openshift
```
It will remove the project `planner-services` from MiniShift

### Check logs from services
Use `oc` from MiniShift
```
eval $(minishift oc-env)
```

List out all running services in MiniShift using
```
oc get pods
```
Wait until all pods are in running state and then copy pod name and use following command to see logs
```
oc logs <<pod name>> -f
```

Use `docker` from MiniShift
```
eval $(minishift docker-env)
```

### Service endpoints:
Get minishit IP using following command
```
minishift ip
```
We will use this IP address to reach to services running in minishift

You can visit database running in minishift by visiting
> psql -h `minishift ip` -U postgres -d postgres -p 32000

WIT service(Work Item Tracker) is running at `minishift ip`:30000

AUTH service is running at `minishift ip`:31000

##### Questions? Errors? Please open issues on https://github.com/fabric8-services/fabric8-wit/issues if already not exist.

You can join our [channel](https://chat.openshift.io/developers/channels/fabric8-planner)
