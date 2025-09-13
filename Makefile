.PHONY: help build test run clean docker-build docker-up docker-down deploy benchmark

build:
	@echo "Building KubeSight..."
	go mod tidy
	go build -o bin/kubesight-server cmd/server/main.go
	@echo "Build complete!"

build-worker:
	@echo "Building worker..."
	go build -o bin/kubesight-worker cmd/worker/main.go
	@echo "Worker build complete!"

run:
	@echo "Starting KubeSight server..."
	./bin/kubesight-server

run-dev:
	@echo "Starting KubeSight in development mode..."
	export KAFKA_BROKERS=localhost:9092 && \
	export SERVER_PORT=8080 && \
	go run cmd/server/main.go

docker-build:
	@echo "Building Docker images..."
	docker build -f deployments/docker/Dockerfile.server -t kubesight:latest .
	@echo "Docker images built!"

docker-up:
	@echo "Starting all services..."
	docker compose up -d
	@echo "Services started!"
	@echo "Dashboard: http://localhost:8080"
	@echo "Kafka UI: http://localhost:8081"
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	@echo "Prometheus: http://localhost:9090"

docker-down:
	@echo "Stopping all services..."
	docker-compose down -v
	@echo "Services stopped!"

docker-logs:
	docker-compose logs -f

fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Code formatted!"

mod-tidy:
	@echo "Tidying modules..."
	go mod tidy
	@echo "Modules tidied!"

setup-grafana:
	@echo "Setting up Grafana..."
	./scripts/grafana_setup.sh
	@echo "Grafana setup complete!"

generate-query:
	@echo "Generating sample queries..."
	./scripts/generate_query.sh
	@echo "Sample queries generated!"

k8s-deploy:
	@echo "Deploy K8S For Namespace-kubesight-system"
	chmod +x ./deploy_k8s.sh
	./deploy_k8s.sh
	@echo "Run Make setup-grafana Command For Monitoring And localhost:8080 For UI Dashboard."

logs:
	@echo "Application logs:"
	docker-compose logs -f kubesight

stats:
	@echo "Current statistics:"
	curl -s http://localhost:8080/api/v1/stats | jq

health:
	@echo "Health check:"
	curl -s http://localhost:8080/health | jq

clean:
	@echo "Cleaning up..."
	rm -rf bin/
	go clean -cache
	@echo "Cleanup complete!"
