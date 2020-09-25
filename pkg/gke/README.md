# GKE syncer

**Important:** Variables enclosed by `[[[]]]` should be replaced.

This syncer is intended to run in your Kubernetes Cluster on GKE and sync DNS records on Cloudflare with your nodes IPs.

The deployment config is like below.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubernetes-cloudflare-syncer
  labels:
    app: kubernetes-cloudflare-syncer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kubernetes-cloudflare-syncer
  template:
    metadata:
      labels:
        app: kubernetes-cloudflare-syncer
    spec:
      serviceAccountName: kubernetes-cloudflare-syncer
      containers:
      - name: kubernetes-cloudflare-syncer
        image: ghcr.io/matts966/kubernetes-cloudflare-syncer/gke:latest
        args:
        - --dns-name=[[[kubernetes.example.com]]]
        env:
        - name: CF_API_KEY
          valueFrom:
            secretKeyRef:
              name: cloudflare
              key: api-key
        - name: CF_API_EMAIL
          valueFrom:
            secretKeyRef:
              name: cloudflare
              key: email
```

**Important:** Make sure to replace `--dns-name=kubernetes.example.com`.

This syncer needs two types of permissions:
1. talk to cloudflare and update DNS
2. get a list of nodes in the cluster and read their IP

The former requires just the API keys from cloudflare. We can store them as secret in the cluster by running:

`kubectl create secret generic cloudflare --from-literal=email=YOUR_CLOUDFLARE_ACCOUNT_EMAIL_ADDRESS_HERE --from-literal=api-key=YOUR_CLOUDFLARE_GLOBAL_API_KEY_HERE`

For the latter we create a `clusterrolebinding` in our cluster by running:

`kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user YOUR_EMAIL_ADDRESS_HERE`

**Important:** Make sure this is the same E-Mail as you use for running kubectl.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubernetes-cloudflare-syncer
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubernetes-cloudflare-syncer
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubernetes-cloudflare-syncer-viewer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kubernetes-cloudflare-syncer
subjects:
- kind: ServiceAccount
  name: kubernetes-cloudflare-syncer
  namespace: default
```

Applying all configs by running:

`kubectl apply -f ./manifests`
