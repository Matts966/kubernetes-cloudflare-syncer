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
  nodeSelector:
    kubernetes.io/hostname: gaming-hoge-controller
  template:
    metadata:
      labels:
        app: kubernetes-cloudflare-syncer
    spec:
      serviceAccountName: kubernetes-cloudflare-syncer
      containers:
      - name: kubernetes-cloudflare-syncer
        image: docker.pkg.github.com/matts966/kubernetes-cloudflare-syncer/gcloud
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

**Important:** Make sure to replace `--projects=your-project` and `--dns-name=kubernetes.example.com`. You can use `--filters` flag to filter instances. Use selector such as `nodeSelector` to schedule the syncer on Google Compute Engine by replacing `kubernetes.io/hostname: gaming-hoge-controller`.


This syncer needs two types of permissions:
1. talk to cloudflare and update DNS
2. get a list of nodes using GCP API and read their IP

The former requires just the API keys from cloudflare. We can store them as secret in the cluster by running:

`kubectl create secret generic cloudflare --from-literal=email=YOUR_CLOUDFLARE_ACCOUNT_EMAIL_ADDRESS_HERE --from-literal=api-key=YOUR_CLOUDFLARE_GLOBAL_API_KEY_HERE`

For the latter you can setup credentials on GCP by commands below.

```bash
export PROJECT_ID=[[[your-project]]]
gcloud iam service-accounts create iplister \
  --display-name "SA to list instances' ip addresses"
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:iplister@$PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/compute.viewer
gcloud iam service-accounts keys create gcp_credentials.json \
  --iam-account example-sa@$PROJECT_ID.iam.gserviceaccount.com
kubectl create secret generic iplister-gcp-cred \
  --from-file=gcp_credentials.json=./gcp_credentials.json
rm ./gcp_credentials.json
```

Applying the config by running:

`kubectl apply -f ./manifests`
