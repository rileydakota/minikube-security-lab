# Kubernetes Security lab with Minikube!

In this lab, you will learn how to:

- Build and Deploy a containerized application to Kubernetes
- Expose your web service to make it accessible outside of the cluster
- Discover some malicious activity in our cluster!
- Add some basic security guardrails
- Exploit a vulnerable application running on a cluster and simulate an attacker!


### Getting Started

The lab assumes you are using MacOS. To start, make sure you have brew installed! 

The lab uses [Taskfiles](https://taskfile.dev) to automate management of the lab, install it:

`brew install go-task/tap/go-task`

You will also need either docker or podman as a container runtime, please install one of the following:

- Docker Desktop
- Podman


With all of that out of the way, start the lab by running the following:

```bash
task labs:start
```

This may take a few minutes to run.

# Step zero - getting familiar with kubectl (kube-cuddle, kube-control, or kube-C-T-L)

Most interactions with K8s are performed via kubectl, a CLI tool designed for interacting with the Kubernetes API Server. We can use the built in whoami command to figure out what authentication context we are running in:

```bash
kubectl auth whoami
```

Kubectl relies on configurations (and sometimes credentials) in a file known as the `kubeconfig` file. Lets take a look at ours:

```bash
cat ~/.kube/config
```

As part of the lab setup, a client certificate and key were configured in your Kubeconfig to be able to access minikube. This is okay for a lab environment, but not ideal in the real world. Kubernetes has a number of options for [authentication](https://kubernetes.io/docs/reference/access-authn-authz/authentication/), and the managed Kubernetes offerings use their respective cloud provider credentials for authentication ([GKE](https://cloud.google.com/kubernetes-engine/docs/how-to/api-server-authentication), [EKS](https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html), [AKS](https://learn.microsoft.com/en-us/azure/aks/enable-authentication-microsoft-entra-id#access-your-enabled-cluster)).

Kubernetes applications and configurations are expressed via "Resources". We can view what types of resources are available on the cluster using the api-resources command:

```bash
kubectl api-resources
```

Seem overwhelming? Don't worry, we will only need a small number of these in the lab. Each of these resources has its own configurations. Lucky for us, kubectl has a built in command that allows us to retrieve the fields and associated documentation. Lets try this for the `pod` resource type:

```bash
kubectl explain pod
```

```bash
kubectl explain pod.spec
```


Lets use some basic commands to see whats on our running cluster:

```bash
kubectl get namespaces
kubectl get pods
kubectl get networkpolicies
kubectl get validatingadmissionpolicy
```

--------

# Step one - Deploy our awesome api service

Congratulations, you have just finished developing your prototype AWESOME API, and are now ready to deploy it straight into production, right!?

To deploy our image into Kubernetes, we now need to package it up into a container. Start by browsing to the `student_resources/our-awesome-api` directory, where the source code for our awesome api. 

Create a file named `Dockerfile` in that directory. Dockerfile consist of a list of instructions on how to build, package, and run your application. More information here: https://docs.docker.com/reference/dockerfile/

Use the following guidelines to build you 
 Use the `golang:1.24-alpine` base image, copy the `go.mod` and `main.go` files into the container, and execute the `go build -o api` command to build our application and output. Finally, add an `ENTRYPOINT` that points to the binary you output. 

<details>
  <summary>Answer</summary>
  
  ```Dockerfile
  #Dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY main.go .
RUN CGO_ENABLED=0 go build -o api -v


EXPOSE 8080
ENTRYPOINT ["./api"]
  ```
</details>

Next, we need to actually build our container and make it available to the cluster. For the purpose of the lab, we are going to utilize the docker daemon that exists in the Minikube cluster itself. In the real world, we would publish to a container registry such as [Google Artifact Registry](), [Elastic Container Registry](), [GitHub Container Registry](), or other options. This allows us to build a container image and make it available to the cluster without having to host a container registry. Minikube provides a command to make this easy for us:

```bash
eval $(minikube docker-env)
```

An easy way to check if this succeeded is to execute a docker command and verify that you are seeing containers/images related to the minikube cluster
```bash
docker image ls
...
...
...
...
registry.k8s.io/kube-apiserver              v1.32.0    2b5bd0f16085   5 months ago    93.9MB
registry.k8s.io/kube-scheduler              v1.32.0    c3ff26fb59f3   5 months ago    67.9MB
registry.k8s.io/kube-controller-manager     v1.32.0    a8d049396f6b   5 months ago    87.2MB
registry.k8s.io/kube-proxy                  v1.32.0    2f50386e20bf   5 months ago    97.1MB
calico/kube-controllers                     v3.29.1    32c335fdb9d7   5 months ago    78.5MB
calico/cni                                  v3.29.1    e5ca62af4ff6   5 months ago    210MB
calico/node                                 v3.29.1    680b8c280812   5 months ago    398MB
registry.k8s.io/etcd                        3.5.16-0   7fc9d4aa817a   8 months ago    142MB
registry.k8s.io/coredns/coredns             v1.11.3    2f6c962e7b83   9 months ago    60.2MB
registry.k8s.io/pause                       3.10       afb61768ce38   11 months ago   514kB
gcr.io/k8s-minikube/storage-provisioner     v5         ba04bb24b957   4 years ago     29MB
```

Finally, lets build and tag our container image:

```bash
docker build . -t awesome-api:v1.0
```

Now that our image is built and published to the Docker runtime within our cluster, we need to create the resources that represent our application on the minikube cluster. Kubectl offers two methods for creating and managing resources on our clusters:

**Imperative**: We can directly create resources using kubectl commands such as run, expose, create, update, or delete. See https://kubernetes.io/docs/tasks/manage-kubernetes-objects/imperative-command/ in the Kubernetes docs for more info
**Declarative**: We can express our desired resource configurations in manifest files, and utilize `kubectl apply` to manage the lifecycle of those objects. Kubectl uses an annotation `kubectl.kubernetes.io/last-applied-configuration` to manage the state, and determine when updates are needed between the local configuration and actual configuration.

For this step, we will utilize declarative configuration by creating a manifest file to represent our service. We need to create:

A Deployment resource


<details>
  <summary>**Answer**</summary>
    ```yaml
apiVersion: v1
kind: Service
metadata:
  name: awesome-api
spec:
  type: ClusterIP
  ports:
  - port: 8080
    targetPort: 8080
  selector:
    app: awesome-api
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: awesome-api
spec:
  replicas: 1
  selector:
    matchLabels:
      app: awesome-api
  template:
    metadata:
      labels:
        app: awesome-api
    spec:
      containers:
      - name: awesome-api
        image: awesome-api:v1.0
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
    ```
</details>




# Step two - oh no, evil!
# Step three - container hardening
# Step four - exploiting a vulnerable service
# Step five - control plane hardening


    


