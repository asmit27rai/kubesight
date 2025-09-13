#!/bin/bash

# deploy_kubesight_k8s.sh
#
# Automates full Kubernetes deployment of KubeSight and supporting stack.
# After running this script, simply run `make setup-grafana` and open Grafana.

set -e

echo "🚀 Starting full KubeSight Kubernetes deployment..."

# 1. Build Go binaries
echo "1. Building KubeSight server and worker..."
make build
make build-worker

# 2. Build Docker images
echo "2. Building Docker images..."
docker build -f deployments/docker/Dockerfile.server -t kubesight:latest .
docker build -f deployments/docker/Dockerfile.worker -t kubesight-worker:latest .

# 3. Detect and load images into cluster
echo "3. Loading images into cluster..."
if command -v kind >/dev/null 2>&1; then
  kind load docker-image kubesight:latest
  kind load docker-image kubesight-worker:latest
elif command -v minikube >/dev/null 2>&1; then
  minikube image load kubesight:latest
  minikube image load kubesight-worker:latest
else
  echo "⚠️  Neither kind nor minikube detected; ensure images are available in your cluster"
fi

# 4. Create namespace
echo "4. Creating namespace kubesight-system..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: kubesight-system
EOF

# 5. Deploy Kafka and Zookeeper via Helm
echo "5. Installing Kafka (Helm chart)..."
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
helm upgrade --install kafka bitnami/kafka \
  --namespace kubesight-system \
  --set replicationFactor=1,numPartitions=3,persistence.enabled=false,zookeeper.persistence.enabled=false

# 6. Deploy Redis via Helm
echo "6. Installing Redis (Helm chart)..."
helm upgrade --install redis bitnami/redis \
  --namespace kubesight-system \
  --set auth.enabled=false,persistence.enabled=false

# 7. Deploy kube-state-metrics
echo "7. Installing kube-state-metrics..."
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm upgrade --install kube-state-metrics prometheus-community/kube-state-metrics \
  --namespace kubesight-system \
  --set service.port=8080

# 8. Deploy Prometheus
echo "8. Installing Prometheus..."
helm upgrade --install prometheus prometheus-community/prometheus \
  --namespace kubesight-system \
  --set server.persistentVolume.enabled=false,alertmanager.enabled=false,pushgateway.enabled=false

# 9. Deploy Grafana
echo "9. Installing Grafana..."
helm upgrade --install grafana bitnami/grafana \
  --namespace kubesight-system \
  --set admin.password=admin123,persistence.enabled=false

# 10. Apply KubeSight ConfigMap
echo "10. Applying KubeSight ConfigMap and Deployments..."
kubectl apply -f deployments/k8s/configmap.yaml
kubectl apply -f deployments/k8s/deployment.yaml
kubectl apply -f deployments/k8s/service.yaml
kubectl apply -f deployments/k8s/worker-deployment.yaml

# 11. Wait for all pods to be ready
echo "11. Waiting for pods to be ready..."
kubectl rollout status deployment/kubesight-server -n kubesight-system
kubectl rollout status deployment/kubesight-worker -n kubesight-system
kubectl rollout status statefulset/kafka-zookeeper -n kubesight-system || true
kubectl rollout status deployment/redis-master -n kubesight-system || true
kubectl rollout status deployment/kube-state-metrics -n kubesight-system
kubectl rollout status deployment/prometheus-server -n kubesight-system
kubectl rollout status deployment/grafana -n kubesight-system

echo "✅ KubeSight and all dependencies deployed successfully!"
echo ""
echo "Next steps:"
echo " 1. Run: make setup-grafana"
echo " 2. Open Grafana: http://localhost:3000 (admin/admin123)"
echo " 3. Open KubeSight dashboard: http://localhost:8080"
echo ""
echo "Enjoy your real-world Kubernetes observability demo!"
