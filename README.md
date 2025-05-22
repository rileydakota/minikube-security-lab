# Kubernetes Security Lab with Minikube!

In this lab, you will learn how to:

- Build and Deploy a containerized application to Kubernetes
- Expose your web service to make it accessible outside of the cluster
- Discover some malicious activity in our cluster!
- Add some basic security guardrails
- Exploit a vulnerable application running on a cluster and simulate an attacker!


### Getting Started

The lab uses [Taskfiles](https://taskfile.dev) to automate management of the lab.

You will also need either docker or podman as a container runtime, please install one of the following:

- Docker Desktop
- Podman

With all of that out of the way, start the lab by running the following:

```bash
task labs:start
```

This may take a few minutes to run.

# Step Zero - Getting Familiar with kubectl (kube-cuddle, kube-control, or kube-C-T-L)

Most interactions with K8s are performed via kubectl, a CLI tool designed for interacting with the Kubernetes API Server. We can use the built-in whoami command to figure out what authentication context we are running in:

```bash
kubectl auth whoami
```

Kubectl relies on configurations (and sometimes credentials) in a file known as the `kubeconfig` file. Let's take a look at ours:

```bash
cat ~/.kube/config
```

As part of the lab setup, a client certificate and key were configured in your Kubeconfig to be able to access minikube. This is okay for a lab environment, but not ideal in the real world. Kubernetes has a number of options for [authentication](https://kubernetes.io/docs/reference/access-authn-authz/authentication/), and the managed Kubernetes offerings use their respective cloud provider credentials for authentication ([GKE](https://cloud.google.com/kubernetes-engine/docs/how-to/api-server-authentication), [EKS](https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html), [AKS](https://learn.microsoft.com/en-us/azure/aks/enable-authentication-microsoft-entra-id#access-your-enabled-cluster)).

Kubernetes applications and configurations are expressed via "Resources". We can view what types of resources are available on the cluster using the api-resources command:

```bash
kubectl api-resources
```

Seem overwhelming? Don't worry, we will only need a small number of these in the lab. Each of these resources has its own configurations. Lucky for us, kubectl has a built-in command that allows us to retrieve the fields and associated documentation. Let's try this for the `pod` resource type:

```bash
kubectl explain pod
```

We can index into specific fields of the documnetation of a resource:

```bash
kubectl explain pod.spec
```

Let's use some basic commands to see what's on our running cluster:

```bash
kubectl get pods
```

We haven't deployed anything to our cluster yet. Kubernetes deploys a default namespace known as `kube-system`, that is reserved for cluster components, such as the API server. 

```bash
kubectl get pods -n kube-system
```

Pick one of the running pods, and describe it to see detailed information!

```bash
kubectl describe pod PODNAME -n kube-system
```

--------

# Step One - Deploy Our Awesome API Service

Congratulations, you have just finished developing your prototype AWESOME API, and are now ready to deploy it straight into production, right!?

To deploy our image into Kubernetes, we now need to package it up into a container. Start by browsing to the `student_resources/our-awesome-api` directory, where the source code for our awesome api. 

Create a file named `Dockerfile` in that directory. Dockerfiles consist of a list of instructions on how to build, package, and run your application. More information here: https://docs.docker.com/reference/dockerfile/

Use the following guidelines to build your container: Use the `golang:1.24-alpine` base image, copy the `go.mod` and `main.go` files into the container, and execute the `go build -o api` command to build our application and output. Finally, add an `ENTRYPOINT` that points to the binary you output. 

<details>
  <summary>Answer</summary>
  
  ```Dockerfile
  #Dockerfile
FROM golang:1.24-alpine
WORKDIR /app
COPY go.mod .
COPY main.go .
RUN CGO_ENABLED=0 go build -o api -v


EXPOSE 8080
ENTRYPOINT ["./api"]
  ```
</details>

Next, we need to actually build our container and make it available to the cluster. For the purpose of the lab, we are going to utilize the docker daemon that exists in the Minikube cluster itself. In the real world, we would publish to a container registry such as Google Artifact Registry, Elastic Container Registry, GitHub Container Registry, or other options. This allows us to build a container image and make it available to the cluster without having to host a container registry. Minikube provides a command to make this easy for us:

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

Finally, let's build and tag our container image:

```bash
docker build . -t awesome-api:v1.0
```
-----------

Now that our image is built and published to the Docker runtime within our cluster, we need to create the resources that represent our application on the minikube cluster. Kubectl offers two methods for creating and managing resources on our clusters:

**Imperative**: We can directly create resources using kubectl commands such as run, expose, create, update, or delete. See https://kubernetes.io/docs/tasks/manage-kubernetes-objects/imperative-command/ in the Kubernetes docs for more info

**Declarative**: We can express our desired resource configurations in manifest files, and utilize `kubectl apply` to manage the lifecycle of those objects. Kubectl uses an annotation `kubectl.kubernetes.io/last-applied-configuration` to manage the state, and determine when updates are needed between the local configuration and actual configuration.

For this step, we will utilize declarative configuration by creating a manifest file to represent our service. We need to create:

A Deployment resource that contains a pod template for our `awesome-api:v1.0` container
The Deployment resource should:
 - Use an imagePullPolicy of `never` (This is a hack/workaround due to not using an actual registry)
 - Run a single replica
ß
The Service Resource should:
 - have a Service type of NodePort so we can access our API externally from the cluster
 - expose port 80, but target port 8080
 - Route traffic to pods with the labels of `app: awesome-api`

Create this in a file named `awesome-api-service.yml`

<details>
  <summary>Answer</summary>
  
```yaml
apiVersion: v1
kind: Service
metadata:
  name: awesome-api
spec:
  type: NodePort
  ports:
  - port: 80
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

With our `awesome-api-service.yml` file created, let's deploy it! 


```bash
kubectl apply -f awesome-api-service.yml & kubectl get pods -w
```

The `-w` flag lets you watch resource changes in realtime. When your awesome-api pod flips to `Running` status, press ctrl+c to stop watching and return to your regular terminal.

When it comes to containerized (and modern/cloud native in general) applications, logs should be [emitted via stdout](https://12factor.net/logs), where other services can collect, enrich, and route them. Kubernetes natively provides the ability to observe a running container's stdout messages using kubectl. Let's observe the logs of our newly running API service:

```bash
kubectl logs -l app=awesome-api -f
```

Notice anything interesting about the logs being produced? The endpoints being requested? Pods in Kubernetes are also granted an individual IP address. Let's see if we can figure out what is querying our API endpoint with some very interesting queries. Make note of the source IP in our logs that the request is coming from. Kubectl is incredibly powerful, and supports tons of [different output options](https://kubernetes.io/docs/reference/kubectl/quick-reference/#formatting-output), we can use the following kubectl command to list all pods, with their associated namespaces and IP addresses:

```bash
kubectl get pods -A -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,IP:.status.podIP
```

What pod is making requests to our API endpoint? What namespace is it in?


<details>
  <summary>Answer</summary>
  
There is an "evil-pod" pod residing in the "evil-here" namespace that is making the calls to our API endpoint, attempting to enumerate URL paths that might resolve to something interesting: 

```bash
kubectl get pods -A -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,IP:.status.podIP

NAMESPACE     NAME                                       IP
default       awesome-api-979588db-564tq                 10.244.120.77
evil-here     evil-pod                                   10.244.120.76  <- SUSPICIOUS 
...
...
...
```
</details>

--------------

## Step 2 - Oh No, Evil!

In the previous lab step, we discovered an "evil-pod" that appears to be a compromised container in our cluster trying to interact with our API. We should probably do something about this! Kubernetes by default has a "flat network" for pods, meaning that workloads on different namespaces or nodes can communicate with each other.

Kubernetes supports the [NetworkPolicy](https://kubernetes.io/docs/concepts/services-networking/network-policies/), which allows you to control network traffic flow between workloads. For a NetworkPolicy to actually take effect, you must be using a Container Network Interface (CNI) that properly supports NetworkPolicies. We have launched the Minikube cluster in the lab with [Calico](https://docs.tigera.io/calico/latest/about), which supports both the base NetworkPolicy resource and the more powerful Calico NetworkPolicy resource. 

With that in mind, let's use a NetworkPolicy to isolate the evil-pod and evil-namespace, and prevent further communication to other pods. 

1. Start by creating the file `namespace-net-isolation.yml`
2. We want to isolate all local egress traffic from the pod, and only allow internet bound traffic (so we can observe any potential C2 activity)
3. This should be scoped for the `evil-here` namespace
4. To avoid accidentally applying this elsewhere, let's explicitly specify the namespace in the manifest
5. Consider blocking IPv6 and the Cloud Instance Metadata Service too :smile:

<details>
  <summary>Answer</summary>
  
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  namespace: evil-here
  name: deny-private-egress
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - to:
    - ipBlock:
        cidr: 0.0.0.0/0
        except:
        - 10.0.0.0/8
        - 172.16.0.0/12
        - 192.168.0.0/16
        - 169.254.169.254/32
    - ipBlock:
        cidr: "::/0"
        except:
        - fc00::/7
        - fe80::/10
        - fd00::/8
```
</details>

Once we have created our NetworkPolicy, let's apply it:

```bash
kubectl apply -f namespace-net-isolation.yml
```

Now let's check out logs for our app, do you still see requests coming in?

```bash
kubectl logs -l app=awesome-api -f --since 5m
```

We want to further observe the activity of our malicious container. In our lab setup, we installed [Tetragon](https://tetragon.io/docs/overview/) onto our cluster. Tetragon is a Kubernetes aware runtime security tool. Tetragon relies on a technology known as the Extended Berkeley Packet Filter, or eBPF, that allows programs to be run directly within the Linux kernel, without modifying the code. Tetragon has insight into the [system calls (syscalls)](https://man7.org/linux/man-pages/man2/syscalls.2.html) that are being made by applications and processes in our cluster. Let's create a TracingPolicy to observe network traffic in our cluster, and potentially find the C2 being utilized. Create a file named `tetragon-network-traffic.yml` with the following: 


```yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: "monitor-network-activity-outside-cluster-cidr-range"
spec:
  kprobes:
  - call: "tcp_connect"
    syscall: false
    args:
    - index: 0
      type: "sock"
    selectors:
    - matchArgs:
      - index: 0
        operator: "NotDAddr"
        values:
        - 127.0.0.1
```

Apply it with the following command:

```bash
kubectl apply -f tetragon-network-traffic.yml
```

Now we can observe the TCP network traffic related to the `evil-here` namespace:

```bash
kubectl exec -ti -n kube-system ds/tetragon -c tetragon -- tetra getevents -o compact --namespace evil-here
```

For now, let's evict the evil-pod from our system:

```bash
kubectl delete pod evil-pod -n evil-here
```

------------

# Step Three - exploiting a vulnerable service

Now that we've deployed our API and addressed the malicious pod, let's explore another security aspect: attacking services hosted on Kubernetes.

Our awesome-api service is exposed outside our cluster through a `NodePort` service. While we can use kubectl to get the `ClusterIP` of our service, that other services and the evil pod were using to interact with the API, to access it externally, we need to know the IP address of the node and ports it is exposed on. Minikube has a command for that:

```bash
minikube service list
```   

This command will show a list of all services and their respective URLs. Note the IP address and port number assigned to our awesome-api service.

Looking through the the source code of our awesome-api, you might notice that our awesome-api has a `/getPhoto` endpoint that accepts a path parameter. This endpoint was intended to serve images from a specific directory, but it has a critical vulnerability: path traversal. This type of vulnerability allows attackers to read files from anywhere on the filesystem by manipulating the path parameter.

Let's exploit this vulnerability to steal a Kubernetes service account token. Typically the token for this account is mounted at a specific path inside the container at the following path `/var/run/secrets/kubernetes.io/serviceaccount/token`:

```bash
curl YOUR_NODE_IP:PORT/getPhoto?path=/var/run/secrets/kubernetes.io/serviceaccount/token
```

Kubernetes Service account tokens are signed JSON Web Tokens (JWTs) issued by the cluster to allow workloads to communicate with the API server. We can examine the jwt using the [https://jwt.io/] website for more information. What service account was this issued for? Are there any other interesting bits of information we can observe?

While the vulnerability in this lab is more academic, the underlying concept applies to the real world. A [recent vulnerability dubbed "Ingress Nightmare"](https://securitylabs.datadoghq.com/articles/ingress-nightmare-vulnerabilities-overview-and-remediation/) involved a similiar mechanism of exploit, where a user could provide a crafted payload to run arbitrary code. The Nginx Ingress service has a service account attached to its pods that have a ClusterRole associated with them, allowing access to all secrets objects in the cluster. Those secrets could then be enumerated for other credentials useful in a lateral movement scenario.  

# Step Four - container hardening

So it turns out that our application had a vulnerability that allowed access to the service account credentials on the pod, which had privileges to perform other privileged actions in our cluster. We should probably fix this.

The first thing we can fix, is to not mount the default service account token if it is not actually needed for our pod. Lets configure the pod template in our awesome-api-deployment to not automount the Service Account Token. Lets configure that on our deployment, and then restart the service to make the change take effect:

<details>
  <summary>Answer</summary>

  in `student_resources/base/awesome-api-service.yml`, under spec > template > spec, add `automountServiceAccountToken: false`. It should look something like the following:

  ```yaml
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
      automountServiceAccountToken: false
      containers:
  ```

  After modfiying the deployment, don't forget to modify it with `kubectl apply -f awesome-api-service.yml`. Modifying the `automountServiceAccountToken` field won't trigger a redeploy of our service. We can also do that using kubectl: 

  ```bash
  kubectl rollout restart deployment awesome-api
  ```
</details>

After configuring the Service Account Token to not automatically mount, lets try the vulnerabilty from the previous step again to retrieve the Service Account Token. Does it work?

Now that we have removed the ServiceAccountToken, we should also remove the code in question that allows the directory traversal vulnerability. Browse to the application code in `student_resources/base/our-awesome-api` and open `main.go`. Find and remove the code that is responsible for the vulnerability. It is okay to remove the feature here :smile:. In the real world, we would likely code migitations that prevent the user from accessing unintended files in the GetPhotos function, or move it to a more exclicit storage backend like GCS.

<details>
  <summary>Answer</summary>

In our-awesome-api, remove the following code:

```go
func getPhotoHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	// Intentionally vulnerable - allows directory traversal
	content, err := ioutil.ReadFile(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(content)
}
```

Don't forget to also remove the code in the main function that references the function we removed:

```go
	http.HandleFunc("/getPhoto", getPhotoHandler)
```

Rebuild our awesome-api application:

```bash
docker build . -t awesome-api:v1.1
```

Modify our awesome-api-service.yml file to reflect the new version and redeploy! Find the old container image tag under spec > template > spec > containers > [0] > image, and update it to the new tag value `awesome-api:v1.1`.

```yaml
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
        image: awesome-api:v1.1  # MODIFY HERE
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
```
</details>




# Step five - control plane hardening

```yaml
apiVersion: admissionregistration.k8s.io/v1 
kind: ValidatingAdmissionPolicy
metadata:
  name: prevent-default-namespace
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups: ["*"]
      apiVersions: ["*"] 
      operations: ["CREATE"]
      resources: ["*"]
  validations:
  - expression: "object.metadata.namespace != 'default'"
    message: "Resource creation is not allowed in the default namespace"

```

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: prevent-default-namespace-binding
spec:
  policyName: prevent-default-namespace
  validationActions: [Deny]
  matchResources:
    namespaceSelector:
      matchLabels:
        kubernetes.io/metadata.name: default

```

```bash
kubectl run busy --image busybox:latest
```

```bash
The pods "busy1" is invalid: : ValidatingAdmissionPolicy 'prevent-default-namespace' with binding 'prevent-default-namespace-binding' denied request: Resource creation is not allowed in the default namespace
```


    


