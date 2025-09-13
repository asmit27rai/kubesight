#!/bin/bash

set -e

echo "Starting full KubeSight Kubernetes deployment..."

echo "1. Building KubeSight server and worker..."
make build
make build-worker

echo "Building Docker images..."
docker build -f deployments/docker/Dockerfile.server -t kubesight:latest .
docker build -f deployments/docker/Dockerfile.worker -t kubesight-worker:latest .

echo "Loading images into cluster..."
if command -v kind >/dev/null 2>&1; then
  kind load docker-image kubesight:latest
  kind load docker-image kubesight-worker:latest
elif command -v minikube >/dev/null 2>&1; then
  minikube image load kubesight:latest
  minikube image load kubesight-worker:latest
else
  echo "Ensure images are available in your cluster"
fi

echo "Creating namespace kubesight-system..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: kubesight-system
EOF

echo "Installing Kafka (Helm chart)..."
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
helm upgrade --install kafka bitnami/kafka \
  --namespace kubesight-system \
  --set replicationFactor=1,numPartitions=3,persistence.enabled=false,zookeeper.persistence.enabled=false

echo "Installing Redis (Helm chart)..."
helm upgrade --install redis bitnami/redis \
  --namespace kubesight-system \
  --set auth.enabled=false,persistence.enabled=false

echo "Installing kube-state-metrics..."
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm upgrade --install kube-state-metrics prometheus-community/kube-state-metrics \
  --namespace kubesight-system \
  --set service.port=8080

echo "Installing Prometheus..."
helm upgrade --install prometheus prometheus-community/prometheus \
  --namespace kubesight-system \
  --set server.persistentVolume.enabled=false,alertmanager.enabled=false,pushgateway.enabled=false

echo "Installing Grafana..."
helm upgrade --install grafana bitnami/grafana \
  --namespace kubesight-system \
  --set admin.password=admin123,persistence.enabled=false

echo "1Applying KubeSight ConfigMap and Deployments..."
kubectl apply -f deployments/k8s/configmap.yaml
kubectl apply -f deployments/k8s/deployment.yaml
kubectl apply -f deployments/k8s/service.yaml
kubectl apply -f deployments/k8s/worker-deployment.yaml

echo "Waiting for pods to be ready..."
kubectl rollout status deployment/kubesight-server -n kubesight-system
kubectl rollout status deployment/kubesight-worker -n kubesight-system
kubectl rollout status statefulset/kafka-zookeeper -n kubesight-system || true
kubectl rollout status deployment/redis-master -n kubesight-system || true
kubectl rollout status deployment/kube-state-metrics -n kubesight-system
kubectl rollout status deployment/prometheus-server -n kubesight-system
kubectl rollout status deployment/grafana -n kubesight-system

echo "KubeSight and all dependencies deployed successfully!"
echo ""
echo "Next steps:"
echo " 1. Run: make setup-grafana"
echo " 2. Open Grafana: http://localhost:3000 (admin/admin)"
echo " 3. Open KubeSight dashboard: http://localhost:8080"
echo ""
