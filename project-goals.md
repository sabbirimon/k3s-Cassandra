# Kubernetes Networking: Service, kube-proxy, and Load Balancing

This comprehensive lab explores Kubernetes networking fundamentals through hands-on implementation. You'll learn how pods communicate within a cluster, how Services direct traffic, and how external access is managed through practical exercises.

![](https://raw.githubusercontent.com/poridhiEng/lab-asset/611799cab62f83188a489ca5e3bfb40c474ff09e/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/kp-arch3-1.drawio.svg)

## Lab Architecture

We'll build a two-tier application:
- **Frontend**: Web server that serves HTTP requests
- **Backend**: Stateful API providing job titles

![](https://raw.githubusercontent.com/poridhiEng/lab-asset/611799cab62f83188a489ca5e3bfb40c474ff09e/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/labarch.drawio.svg)

The frontend calls the backend and displays which pod processed the request.

### Prerequisites:

- Basic Kubernetes knowledge
- Minikube or a Kubernetes cluster. Poridhi Lab provides MultiNode Kubernetes Cluster. You can launch worker nodes according to the need.

  ![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-29.png)

## Part 1: Backend Deployment

### Step 1: Create the Backend Application

First, let's create a simple backend API. Create a directory for your project:

```bash
mkdir k8s-networking-lab
cd k8s-networking-lab
mkdir backend
cd backend
```

Create `server.js`:

```javascript
const express = require('express');
const os = require('os');
const app = express();
const PORT = 5000;

const jobs = [
  'Job 1',
  'Job 2',
  'Job 3',
  'Job 4',
  'Job 5',
];

app.get('/', (req, res) => {
  const randomJob = jobs[Math.floor(Math.random() * jobs.length)];
  res.json({
    job: randomJob,
    pod: os.hostname(),
    podIP: req.connection.localAddress
  });
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`Backend API listening on port ${PORT}`);
  console.log(`Pod: ${os.hostname()}`);
});
```

Create `package.json`:

```json
{
  "name": "backend-api",
  "version": "1.0.0",
  "main": "server.js",
  "dependencies": {
    "express": "^4.18.2"
  },
  "scripts": {
    "start": "node server.js"
  }
}
```

Create `Dockerfile`:

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY server.js ./
EXPOSE 5000
CMD ["npm", "start"]
```

### Step 2: Build and Push Backend Image

```bash
# Build the image
docker build -t <DockerUsername>/backend-api:v1 .
docker push <DockerUsername>/backend-api:v1
```

> NOTE: Make sure to login to Dockerhub

### Step 3: Deploy Backend to Kubernetes

Create `backend-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-deployment
  labels:
    app: backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
      - name: backend
        image: <DockerUsername>/backend-api:v1 # Update with your Docker Hub username
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 5000
          name: http
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "200m"
```

Apply the deployment:

```bash
kubectl apply -f backend-deployment.yaml
```

### Step 4: Verify Backend Deployment

```bash
# Check deployment status
kubectl get deployment backend-deployment

# Check pod details including IP address
kubectl get pod -l app=backend -o wide

# View pod logs
kubectl logs -l app=backend
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-11.png)

> **Note the Pod IP address** - you'll use it later to understand Service networking.


## Part 2: Exposing Backend with ClusterIP Service

In Kubernetes, a Service provides a stable way for different parts of your application to communicate. While pods may restart or change their IP addresses, a Service keeps a consistent, reliable endpoint. This ensures that other components—such as your frontend—can always reach the backend application without having to track pod IP changes.

To allow the frontend pods to communicate with the backend, we will expose the backend deployment using a ClusterIP Service, which makes the backend accessible inside the cluster.

### Step 1: Create Backend Service

Create `backend-service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: backend-service
  labels:
    app: backend
spec:
  type: ClusterIP
  selector:
    app: backend
  ports:
  - name: http
    protocol: TCP
    port: 5000
    targetPort: 5000
```

**What This Definition Means**

- **type: ClusterIP**

  This exposes the backend only inside the cluster. Other pods will be able to reach it, but it won’t be accessible from outside unless additional routing (like an Ingress or NodePort) is configured.

- **selector: app: backend**
  
  The Service automatically forwards traffic to any pod that has the label app=backend.

- **port / targetPort**

  - port: 5000 — the port that other pods will use to reach the Service.

  - targetPort: 5000 — the port the backend container is actually listening on.

Apply the service:

```bash
kubectl apply -f backend-service.yaml
```

### Step 2: Inspect the Service

Check that the Service has been created successfully:

```bash
# View service details
kubectl get svc
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-9.png)


This will show information such as the Service’s cluster-internal IP, port, and type.

## Step 3: DNS Resolution for the Backend Service

Kubernetes Services not only provide stable virtual IPs—they also automatically receive DNS names. This makes it easier for applications to communicate without needing to know or track IP addresses.

![](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-33.png)

### How DNS Works for Services

When you create a Service, Kubernetes automatically registers a DNS record for it.
The fully qualified domain name (FQDN) follows this format:

```bash
<service-name>.<namespace>.svc.cluster.local
```

For example, the backend Service in the `default` namespace is reachable at:

```bash
backend-service.default.svc.cluster.local
```

Because of this, pods can communicate with Services using DNS names instead of hard-coded IP addresses

### CoreDNS: The Cluster DNS Server

Kubernetes uses **CoreDNS** to translate Service names into their corresponding IP addresses.

* CoreDNS runs as a Deployment in the `kube-system` namespace.
* It is exposed internally through a ClusterIP service named **kube-dns**.
* Whenever a pod needs to resolve a Service name, it sends a DNS query to this kube-dns Service.
* CoreDNS processes the query and returns the Service’s ClusterIP address.
  The application then connects to that IP.

You can inspect the kube-dns Service with:

```bash
kubectl get svc -n kube-system kube-dns
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-38.png)


### How Pods Know Where to Send DNS Queries

Kubernetes configures each pod’s `/etc/resolv.conf` file during creation.
This file specifies:

* Which DNS server to use (usually the kube-dns ClusterIP).
* DNS search domains that help expand short names.

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-39.png)


This “search list” means that if a pod tries to resolve just `backend-service`, Kubernetes will automatically try:

1. `backend-service.default.svc.cluster.local`
2. `backend-service.svc.cluster.local`
3. `backend-service.cluster.local`

This allows you to refer to Services simply by name when communicating within the same namespace.

### How DNS Resolution Enables Service Communication

Here’s the typical flow when a frontend pod wants to connect to the backend Service:

1. The frontend pod only knows the Service’s name—not its IP.
2. The application issues a DNS query to CoreDNS.
3. CoreDNS resolves the Service name to its ClusterIP.
4. The application uses that IP to send traffic to the Service.

![](https://raw.githubusercontent.com/poridhiEng/lab-asset/606a24f7487b9e616730cba5767e3e865a1a51dc/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/dns-recovery.drawio.svg)

### Test DNS resolution:

To verify that CoreDNS is working correctly:

```bash
# Check CoreDNS service
kubectl get svc -n kube-system kube-dns

# Test DNS lookup
kubectl run dns-test --rm -i --tty --restart=Never --image=busybox:1.28 -- nslookup backend-service
```

You should see a DNS response containing the backend Service’s ClusterIP.

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-12.png)

### Step 4: Test Backend Connectivity

Now that DNS resolution works, confirm that pods can actually reach the backend Service.


```bash
# Create a test pod
kubectl run curl-test --rm -i --tty --image=curlimages/curl -- sh

# Inside the pod, test the backend
curl http://backend-service:5000
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-13.png)


A successful response from the backend confirms that:

* DNS resolution is working
* The backend Service is reachable
* Network routing inside the cluster is functioning correctly

## **Endpoints and Services**

In Kubernetes, a **Service** does not run a process, does not listen on a port, and does not perform load balancing by itself. A Service is simply a **logical abstraction** that defines *how traffic should be forwarded* to a group of pods.

- There is **no actual load balancer process** running inside the cluster nodes for a Service.
- There is also **no network daemon** binding to the Service’s ClusterIP.

### **Verifying that Services don’t listen on ports**

If you SSH into any cluster node and run:

```bash
netstat -ntlp | grep <service_IP>
netstat -ntlp | grep 5000
```

You will see **no results**.
Why?
Because:

* `<service_IP>` is a **virtual ClusterIP**
* It **does not correspond to any real interface**
* It is **not owned by any process**

The ClusterIP exists only inside the Kubernetes networking layer and is handled by components such as **kube-proxy**, which programs iptables or IPVS rules.

### **So how does a Service know which pods to send traffic to?**

When we create a Service, Kubernetes stores its definition in **etcd**.
At the same time, the **Endpoint Controller** checks the Service’s selector and finds all matching pods.

For example:

```yaml
selector:
  app: backend
```

The Endpoint Controller looks for pods with that label and collects their **Pod IP + port** combinations.

These results are stored in an **Endpoint object**.

![](https://raw.githubusercontent.com/poridhiEng/lab-asset/611799cab62f83188a489ca5e3bfb40c474ff09e/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/endpoint.drawio.svg)

You normally never create Endpoint objects manually — they are automatically created and updated by Kubernetes based on your Services and Pods.

### **Checking existing Services and Endpoints**

Run:

```bash
kubectl get endpoints backend-service
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-10.png)

The output should list the IP addresses of the backend pods. If the endpoints list is empty, it means Kubernetes did not find any pods matching the selector, and the Service will not be able to route traffic.

Here you can see:

* An **Endpoint object** named `backend-service` exists.
* It contains a single endpoint: `10.42.0.5:6000`, which is the IP and port of your backend pod.

### **But how does traffic actually reach the pods?**

Understanding how endpoints are collected is only half of the puzzle. Even though Services don’t run any process, Kubernetes still manages to distribute traffic to all matching pods. Kubernetes accomplishes this by using **kube-proxy**, which sets up:

* iptables rules, or
* IPVS virtual load balancers

These rules intercept traffic destined for the Service’s ClusterIP and **redirect it to one of the endpoints**.

In other words:

> Kubernetes implements a *distributed load balancer* using networking rules programmed on every node.

No single process handles Service load balancing — it is entirely done by the network layer.


## Part 3: Understanding kube-proxy and iptables

Kubernetes Services provide a stable virtual IP address, but there is no actual process “listening” on that Service IP. Instead, Kubernetes uses a networking component called **kube-proxy** to make Service traffic work.

### What Is kube-proxy?

`kube-proxy` runs on every node in your cluster and is responsible for implementing the Kubernetes Service abstraction.

* **Service IPs are virtual**
  When you create a Service, Kubernetes assigns it a ClusterIP, but nothing is physically listening on that IP. It is simply a virtual address.

* **kube-proxy makes Services functional**
  kube-proxy monitors the Kubernetes API for changes in **Services** and **Endpoints**, and programs the node’s networking rules so that any traffic sent to a Service IP is automatically forwarded to one of the corresponding backend Pod IPs.

* **Runs as a DaemonSet**
  Every node runs one instance of kube-proxy, ensuring that service routing works locally on each node.

### How kube-proxy Uses iptables

In many Kubernetes clusters, kube-proxy operates in **iptables mode**, meaning it configures Linux firewall rules to handle Service traffic.

Here’s what happens:

1. kube-proxy installs iptables rules that intercept packets destined for a Service IP.
2. When such traffic arrives on a node, the destination IP is *rewritten* (DNAT) to the IP of one of the backend pods.
3. The packet is then forwarded to that pod.

### kube-proxy in Action: Translating Service IPs to Pod IPs

1. **Services don’t really “exist,” but Kubernetes needs to simulate them**

   ![](https://raw.githubusercontent.com/poridhiEng/lab-asset/611799cab62f83188a489ca5e3bfb40c474ff09e/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/sim1.drawio.svg)

2. **Traffic arriving at the node is intercepted by iptables rules configured by kube-proxy**

    ![](https://raw.githubusercontent.com/poridhiEng/lab-asset/611799cab62f83188a489ca5e3bfb40c474ff09e/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/sim2.drawio.svg)

3. **iptables rewrites the packet’s destination IP to one of the Pod IPs**

    ![](https://raw.githubusercontent.com/poridhiEng/lab-asset/611799cab62f83188a489ca5e3bfb40c474ff09e/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/sim3.drawio.svg)

4. **The packet is forwarded to the selected backend pod**

    ![](https://raw.githubusercontent.com/poridhiEng/lab-asset/611799cab62f83188a489ca5e3bfb40c474ff09e/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/sim4.drawio.svg)


## kube-proxy and iptables rules

iptables is a Linux firewall tool that allows you to configure rules for filtering and manipulating network traffic at the kernel level. Think of iptables as a Mail Sorting System. iptables organizes rules into tables and chains:

![](https://raw.githubusercontent.com/poridhiEng/lab-asset/611799cab62f83188a489ca5e3bfb40c474ff09e/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/lin-kern.drawio.svg)

### iptables Structure: Chains, Rules, Targets

### Chains

Lists of rules applied in order.

Default chains:

* INPUT
* FORWARD
* OUTPUT
* PREROUTING
* POSTROUTING

Kubernetes adds many *custom chains* such as:

* `KUBE-SERVICES`
* `KUBE-SVC-xxxxxx`
* `KUBE-SEP-xxxxxx`

### Rules

Each rule checks:

* Source IP
* Destination IP
* Protocol
* Port
* Connection state
* Interface
* etc.

If the packet matches, **the rule’s target is executed**.

### Targets (Actions)

Common targets:

| Target | Meaning                  |
| ------ | ------------------------ |
| ACCEPT | Allow packet             |
| DROP   | Block packet             |
| DNAT   | Rewrite destination      |
| SNAT   | Rewrite source           |
| JUMP   | Go to another chain      |
| RETURN | Return to previous chain |

![](https://raw.githubusercontent.com/poridhiEng/lab-asset/611799cab62f83188a489ca5e3bfb40c474ff09e/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/kp-arch3-1.drawio.svg)

## Following traffic from a Pod to Service

### Step 1: Inspect iptables Rules

To see how kube-proxy configures iptables, create a privileged debug pod with host networking enabled:

```bash
kubectl run iptables-debug --rm -i --tty --privileged \
  --image=ubuntu \
  --overrides='{"spec": {"hostNetwork": true, "hostPID": true}}' \
  -- bash
```

Inside the debug pod:

```bash
# Install iptables
apt update && apt install -y iptables

# View NAT table chains
iptables-legacy -t nat -L -n -v | grep backend-service
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-14.png)

You can further inspect key iptables chains:

```bash
# View PREROUTING chain
iptables-legacy -t nat -L PREROUTING -n --line-numbers
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-15.png)

```bash
# View KUBE-SERVICES chain
iptables-legacy -t nat -L KUBE-SERVICES -n --line-numbers
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-16.png)

### Step 2: Traffic Flow Visualization

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-40.png)

### Step 3: Find Service-Specific Chains

```bash
# Find the chain for backend-service
iptables-legacy -t nat -L KUBE-SERVICES -n | grep backend-service

# Inspect the service chain
iptables-legacy -t nat -L KUBE-SVC-XXX -n --line-numbers

# Inspect endpoint chain
iptables-legacy -t nat -L KUBE-SEP-XXX -n --line-numbers
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-17.png)

### Step 4: Scale Backend and Observe Changes

```bash
# Exit the debug pod and scale backend
kubectl scale deployment backend-deployment --replicas=3

# Wait for pods to be ready
kubectl get pods -l app=backend -o wide

# Create debug pod again and check iptables
kubectl run iptables-debug --rm -i --tty --privileged \
  --image=ubuntu \
  --overrides='{"spec": {"hostNetwork": true, "hostPID": true}}' \
  -- bash

# Install iptables
apt update && apt install -y iptables

# Now you'll see multiple KUBE-SEP chains
iptables-legacy -t nat -L KUBE-SVC-XXX -n --line-numbers
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-18.png)

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-19.png)

**Load Balancing Visualization:**


![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-41.png)



## Part 4: Frontend Deployment

### Step 1: Create Frontend Application

Navigate back to your project root and create frontend:

```bash
cd ..
mkdir frontend
cd frontend
```

Create `server.js`:

```javascript
const express = require('express');
const axios = require('axios');
const os = require('os');
const app = express();
const PORT = 8080;

const BACKEND_URL = process.env.BACKEND_URL || 'http://backend-service:5000';

app.get('/', async (req, res) => {
  try {
    const response = await axios.get(BACKEND_URL);
    const backendData = response.data;
    
    res.send(`
      <!DOCTYPE html>
      <html>
      <head>
        <title>K8s Networking Demo</title>
        <style>
          body { 
            font-family: Arial, sans-serif; 
            max-width: 800px; 
            margin: 50px auto; 
            padding: 20px;
            background: #f5f5f5;
          }
          .container {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
          }
          h1 { color: #333; }
          .info { 
            background: #e8f4f8; 
            padding: 15px; 
            border-radius: 4px; 
            margin: 10px 0;
          }
          .label { font-weight: bold; color: #555; }
        </style>
      </head>
      <body>
        <div class="container">
          <h1>🚀 Kubernetes Networking Demo</h1>
          <div class="info">
            <p><span class="label">Job Title:</span> ${backendData.job}</p>
            <p><span class="label">Backend Pod:</span> ${backendData.pod}</p>
            <p><span class="label">Backend Pod IP:</span> ${backendData.podIP}</p>
          </div>
          <div class="info">
            <p><span class="label">Frontend Pod:</span> ${os.hostname()}</p>
            <p><span class="label">Client IP:</span> ${req.ip}</p>
          </div>
          <button onclick="location.reload()">🔄 Refresh</button>
        </div>
      </body>
      </html>
    `);
  } catch (error) {
    res.status(500).send(`Error connecting to backend: ${error.message}`);
  }
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`Frontend server listening on port ${PORT}`);
  console.log(`Pod: ${os.hostname()}`);
  console.log(`Backend URL: ${BACKEND_URL}`);
});
```

Create `package.json`:

```json
{
  "name": "frontend-app",
  "version": "1.0.0",
  "main": "server.js",
  "dependencies": {
    "express": "^4.18.2",
    "axios": "^1.6.0"
  },
  "scripts": {
    "start": "node server.js"
  }
}
```

Create `Dockerfile`:

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY server.js ./
EXPOSE 8080
CMD ["npm", "start"]
```

### Step 2: Build and Load Frontend Image

```bash
# Build the image
docker build -t <DockerUsername>/frontend-app:v1 .
docker push <DockerUsername>/frontend-app:v1
```

### Step 3: Deploy Frontend

Create `frontend-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend-deployment
  labels:
    app: frontend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - name: frontend
        image: <DockerUsername>/frontend-app:v1 # Update with your Docker Hub username 
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: BACKEND_URL
          value: "http://backend-service:5000"
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "200m"
```

Apply the deployment:

```bash
kubectl apply -f frontend-deployment.yaml

# Verify deployment
kubectl get pods -l app=frontend -o wide
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-20.png)

## Part 5: NodePort Service

### Step 1: Create NodePort Service

Create `frontend-service-nodeport.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: frontend-service
  labels:
    app: frontend
spec:
  type: NodePort
  selector:
    app: frontend
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
    # nodePort: 32073  # Optional: specify a port in range 30000-32767
```

Apply the service:

```bash
kubectl apply -f frontend-service-nodeport.yaml

# Check the assigned NodePort
kubectl get service frontend-service
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-21.png)

```bash
kubectl get endpoints frontend-service
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-22.png)

```bash
iptables-legacy -t nat -L KUBE-SERVICES -n --line-numbers
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-23.png)

```bash
iptables-legacy -t nat -L KUBE-NODEPORTS -n -v --line-numbers | grep 31116
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-24.png)

### Step 2: Access via NodePort

```bash
# Get node IP
kubectl get nodes -o wide
```

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-42.png)

```bash
curl http://<NodeIP>:<NODEPORT>
```

If you need browser access, create a load balancer from the Lab Console pointing to:

* **IP → Master Node Private IP**
* **Port → NodePort**

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-43.png)


Visit the generated URL to access the frontend application.


![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-44.png)



### Step 3: Understand NodePort iptables Rules

NodePort traffic flows through special iptables chains managed by kube-proxy.

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/606a24f7487b9e616730cba5767e3e865a1a51dc/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-47.png)

To Inspect NodePort rules:

```bash
kubectl run iptables-debug --rm -i --tty --privileged \
  --image=ubuntu \
  --overrides='{"spec": {"hostNetwork": true, "hostPID": true}}' \
  -- bash

# Inside the pod
apt update && apt install -y iptables

# Find NodePort chain
iptables-legacy -t nat -L KUBE-NODEPORTS -n --line-numbers

iptables-legacy -t nat -L KUBE-NODEPORTS -n | grep 31116

# Inspect the external traffic chain
iptables-legacy -t nat -L KUBE-EXT-<hash> -n --line-numbers
```

Expected Output:

![alt text](https://raw.githubusercontent.com/poridhiEng/lab-asset/61e69f8ec725ec7c733a5cdaf17e3c606462cc00/Kubernetes%20Labs/Lab%20-%20Kubernetes%20Networking/images/image-25.png)

## Part 6: LoadBalancer Service

### Step 1: Create LoadBalancer Service

Create `frontend-service-lb.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: frontend-service
  labels:
    app: frontend
spec:
  type: LoadBalancer
  selector:
    app: frontend
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
```

Apply the service:

```bash
kubectl apply -f frontend-service-lb.yaml

# Check service status
kubectl get service frontend-service
```

### Step 2: Access via Load Balancer

```bash
curl http://<NodeIP>:<NODEPORT>
```

## Conclusion

This comprehensive lab provides hands-on experience with Kubernetes networking fundamentals through practical implementation. Each section builds on the previous one, allowing you to understand the complete networking stack from pods to external load balancers.