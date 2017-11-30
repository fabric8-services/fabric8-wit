## Start running dependent services on OpenShift

These instructions will help you run your services on OpenShift using MiniShift.

### Prerequisites


[Kedge](kedgeproject.org)

[MiniShift](https://docs.openshift.org/latest/minishift/getting-started/installing.html)

[Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)


### Installation (Linux)

##### Install Kedge

Following steps will download and install Kedge on your machine and put it in your $PATH. For more detailed information please visit kedgeproject.org

```
curl -L https://github.com/kedgeproject/kedge/releases/download/v0.5.1/kedge-linux-amd64 -o kedge
```

Verify installation by running following command, you should get version number.

```
kedge version
```

##### Install Minishift

Make sure you have all prerequisites installed. Please check the list [here](https://docs.openshift.org/latest/minishift/getting-started/installing.html#install-prerequisites)

Download and put `minishift` in your $PATH by following steps [here](https://docs.openshift.org/latest/minishift/getting-started/installing.html#manually)

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

When you want to run fabric8-wit, fabric8-auth and databases on OpenShift use following command
```
make dev-planner-openshift
```
Please enter password when prompted, it is needed in order to make an entry in the `/etc/hosts`.
`minishift ip` gives the IP address on which MiniShift is running. This automation creates a host entry as `minishift.local` for that IP. This domain is whitelisted on fabric8-auth.

This build uses developer account for creating a project called `planner-services`.

Above command then automates the process of running the containers on OpenShift in MiniShift by using Kedge.


## Cleanup
When you want to stop all the services running in MiniShift, use following command
```
make dev-planner-openshift-clean
```
It will remove the project `planner-services` from MiniShift

## Check logs from services
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


##### Reach out to **[/pranavgore09](https://github.com/pranavgore09)** in case things are not working as expected
