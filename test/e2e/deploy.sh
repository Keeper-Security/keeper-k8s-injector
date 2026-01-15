#!/bin/bash
set -e

NAMESPACE=${NAMESPACE:-keeper-system}
CONTEXT=${CONTEXT:-kind-keeper-test}

echo "Creating namespace..."
kubectl --context $CONTEXT create namespace $NAMESPACE --dry-run=client -o yaml | kubectl --context $CONTEXT apply -f -

echo "Generating self-signed certificates..."
TMPDIR=$(mktemp -d)
cd $TMPDIR

# Generate CA
openssl genrsa -out ca.key 2048
openssl req -new -x509 -days 365 -key ca.key -subj "/CN=keeper-injector-ca" -out ca.crt

# Generate server certificate
openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/CN=keeper-injector.$NAMESPACE.svc" -out server.csr

# Create extension file for SAN
cat > extfile.cnf << EOF
subjectAltName = DNS:keeper-injector,DNS:keeper-injector.$NAMESPACE,DNS:keeper-injector.$NAMESPACE.svc,DNS:keeper-injector.$NAMESPACE.svc.cluster.local
EOF

openssl x509 -req -extfile extfile.cnf -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

# Create TLS secret
kubectl --context $CONTEXT -n $NAMESPACE create secret tls keeper-injector-tls \
  --cert=server.crt --key=server.key --dry-run=client -o yaml | kubectl --context $CONTEXT apply -f -

# Get CA bundle for webhook config
CA_BUNDLE=$(base64 -w0 < ca.crt)

cd -
rm -rf $TMPDIR

echo "Deploying webhook..."
cat << EOF | kubectl --context $CONTEXT apply -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: keeper-injector
  namespace: $NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: keeper-injector
rules:
- apiGroups: [""]
  resources: ["pods", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: keeper-injector
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: keeper-injector
subjects:
- kind: ServiceAccount
  name: keeper-injector
  namespace: $NAMESPACE
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keeper-injector
  namespace: $NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: keeper-injector
  template:
    metadata:
      labels:
        app: keeper-injector
    spec:
      serviceAccountName: keeper-injector
      containers:
      - name: webhook
        image: keeper/injector-webhook:dev
        imagePullPolicy: Never
        args:
        - --cert-dir=/etc/webhook/certs
        - --sidecar-image=keeper/injector-sidecar:dev
        - --log-level=debug
        - --log-format=console
        ports:
        - containerPort: 9443
          name: https
        - containerPort: 8081
          name: health
        volumeMounts:
        - name: certs
          mountPath: /etc/webhook/certs
          readOnly: true
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: certs
        secret:
          secretName: keeper-injector-tls
---
apiVersion: v1
kind: Service
metadata:
  name: keeper-injector
  namespace: $NAMESPACE
spec:
  selector:
    app: keeper-injector
  ports:
  - port: 443
    targetPort: 9443
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: keeper-injector
webhooks:
- name: keeper.security.injector
  admissionReviewVersions: ["v1"]
  clientConfig:
    service:
      name: keeper-injector
      namespace: $NAMESPACE
      path: /mutate-pods
    caBundle: $CA_BUNDLE
  rules:
  - operations: ["CREATE"]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
  failurePolicy: Ignore
  sideEffects: None
  namespaceSelector:
    matchExpressions:
    - key: keeper.security/inject
      operator: NotIn
      values: ["disabled"]
EOF

echo "Waiting for deployment to be ready..."
kubectl --context $CONTEXT -n $NAMESPACE rollout status deployment/keeper-injector --timeout=120s

echo "Webhook deployed successfully!"
kubectl --context $CONTEXT -n $NAMESPACE get pods
