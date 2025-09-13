#!/bin/bash

echo "Generating IMPRESSIVE activity for Grafana demo..."

echo "Generating 50,000 metrics burst..."
curl -X POST http://localhost:8080/api/v1/demo/generate \
  -H "Content-Type: application/json" \
  -d '{"count": 50000, "cluster_id": "production-cluster", "namespace": "web-services"}' > /dev/null

sleep 5

echo "⚡ Generating continuous query activity..."
for round in {1..3}; do
  echo "Round $round: Executing query batches..."
  
  for i in {1..20}; do
    curl -s "http://localhost:8080/api/v1/demo/query?type=count_distinct" > /dev/null &
  done
  
  for i in {1..15}; do
    curl -s "http://localhost:8080/api/v1/demo/query?type=percentile" > /dev/null &
  done
  
  for i in {1..10}; do
    curl -s "http://localhost:8080/api/v1/demo/query?type=top_k" > /dev/null &
  done
  
  for i in {1..8}; do
    curl -s "http://localhost:8080/api/v1/demo/query?type=sum" > /dev/null &
    curl -s "http://localhost:8080/api/v1/demo/query?type=average" > /dev/null &
  done
  
  wait
  
  echo "Round $round completed: ~70 queries executed"
  sleep 10
done

echo "Final burst: High-frequency queries..."
for i in {1..50}; do
  curl -s "http://localhost:8080/api/v1/demo/query?type=count_distinct" > /dev/null
  if [ $((i % 10)) -eq 0 ]; then
    echo "Burst progress: $i/50 queries"
  fi
  sleep 0.5
done

echo ""
echo "DEMO QUERY GENERATED!"
echo "Your Grafana dashboard now Ready"
echo "Open http://localhost:3000"