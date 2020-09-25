# GCloud syncer

**Important:** Variables enclosed by `[[[]]]` should be replaced.

This syncer is intended to run in your non-managed Kubernetes Cluster on GCE and sync DNS records on Cloudflare with your nodes IPs.

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
        image: ghcr.io/matts966/kubernetes-cloudflare-syncer/gcloud:latest
        args:
        - --dns-name=[[[kubernetes.example.com]]]
        - --projects=[[[your-project]]]
        - --filters="status = RUNNING"
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
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /var/secrets/google/gcp_credentials.json
          volumeMounts:
          - name: iplister-gcp-cred
            mountPath: /var/secrets/google
      volumes:
      - name: iplister-gcp-cred
        secret:
          secretName: iplister-gcp-cred
```

**Important:** Make sure to replace `--projects=your-project` and `--dns-name=kubernetes.example.com`. You can use `--filters` flag to filter instances

This syncer needs three types of permissions:
1. talk to cloudflare and update DNS
2. get a list of nodes using GCP API and read their IP
3. Watch for the changes for nodes on Cluster

The first one requires just the API keys from cloudflare. We can store them as secret in the cluster by running:

`kubectl create secret generic cloudflare --from-literal=email=YOUR_CLOUDFLARE_ACCOUNT_EMAIL_ADDRESS_HERE --from-literal=api-key=YOUR_CLOUDFLARE_GLOBAL_API_KEY_HERE`

For the second one, you can setup credentials on GCP by commands below.

```bash
export PROJECT_ID=[[[your-project]]]
gcloud iam service-accounts create iplister \
  --display-name "SA to list instances' ip addresses"
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:iplister@$PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/compute.viewer
gcloud iam service-accounts keys create gcp_credentials.json \
  --iam-account iplister@$PROJECT_ID.iam.gserviceaccount.com
kubectl create secret generic iplister-gcp-cred \
  --from-file=gcp_credentials.json=./gcp_credentials.json
rm ./gcp_credentials.json
```

For the last one, we create the role below.

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

Applying the config by running:

`kubectl apply -f ./manifests`
