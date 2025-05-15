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

`task labs:start`

This may take a few minutes to run.

# Step zero - getting familiar with kubectl (kube-cuddle, kube-control, or kube-C-T-L)

Most interactions with K8s are performed via kubectl, a CLI tool designed for interacting with the Kubernetes API Server. We can use the built in whoami command to figure out what authentication context we are running in:

`kubectl auth whoami`

Kubectl relies on configurations (and sometimes credentials) in a file known as the `kubeconfig` file. Lets take a look at ours:

`cat ~/.kube/config`

As part of the lab setup, a client certificate and key were configured in your Kubeconfig to be able to access minikube. This is okay for a lab environment, but not ideal in the real world. Kubernetes has a number of options for [authentication](https://kubernetes.io/docs/reference/access-authn-authz/authentication/), and the managed Kubernetes offerings use their respective cloud provider credentials for authentication.

Kubernetes applications and configurations are expressed via "Resources". We can view what types of resources are available on the cluster using the api-resources command:

`kubectl api-resources`

Seem overwhelming? Don't worry, we will only need a small number of these in the lab. Each of these resources has its own configurations. Lucky for us, kubectl has a built in command that allows us to retrieve the fields and associated documentation. Lets try this for the `pod` resource type:

`kubectl explain pod`
`kubectl explain pod.spec`


Lets use some basic commands to see whats on our running cluster:

`kubectl get namespaces`
`kubectl get pods`
`kubectl get networkpolicies`
`kubectl get validatingadmissionpolicy`





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


# Step two - oh no, evil!
# Step three - container hardening
# Step four - exploiting a vulnerable service
# Step five - control plane hardening


    


