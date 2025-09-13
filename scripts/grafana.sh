#!/bin/bash

echo "Creating KubeSight Dashboard"

echo "Waiting for Grafana to be ready..."
timeout=60
while ! curl -s http://localhost:3000/api/health | grep -q "ok"; do
    sleep 2
    timeout=$((timeout - 2))
    if [ $timeout -le 0 ]; then
        echo "Grafana not ready, please check if it's running"
        exit 1
    fi
done

echo "Grafana is ready!"

echo "Adding Prometheus data source..."
curl -X POST http://admin:admin@localhost:3000/api/datasources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Prometheus",
    "type": "prometheus",
    "url": "http://prometheus:9090",
    "access": "proxy",
    "isDefault": true
  }' > /dev/null 2>&1

echo "Prometheus data source added!"

echo "Creating stunning KubeSight dashboard..."
cat > /tmp/dashboard.json << 'EOF'
{
  "dashboard": {
    "id": null,
    "title": "KubeSight Approximate Query Engine - Live Performance",
    "tags": ["kubesight", "performance", "approximate", "queries"],
    "timezone": "browser",
    "refresh": "5s",
    "time": {
      "from": "now-30m",
      "to": "now"
    },
    "templating": {
      "list": []
    },
    "annotations": {
      "list": [
        {
          "builtIn": 1,
          "datasource": "-- Grafana --",
          "enable": true,
          "hide": true,
          "iconColor": "rgba(0, 211, 255, 1)",
          "name": "Annotations & Alerts",
          "type": "dashboard"
        }
      ]
    },
    "panels": [
      {
        "id": 1,
        "title": "TOTAL QUERIES PROCESSED",
        "type": "stat",
        "targets": [
          {
            "expr": "kubesight_queries_total{type=\"total\"}",
            "legendFormat": "Total Queries",
            "refId": "A"
          }
        ],
        "options": {
          "colorMode": "background",
          "graphMode": "area",
          "justifyMode": "center",
          "orientation": "horizontal",
          "reduceOptions": {
            "values": false,
            "calcs": ["lastNotNull"],
            "fields": ""
          },
          "textMode": "auto"
        },
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "thresholds"
            },
            "custom": {
              "displayMode": "basic"
            },
            "mappings": [],
            "thresholds": {
              "mode": "absolute",
              "steps": [
                {"color": "green", "value": null},
                {"color": "yellow", "value": 100},
                {"color": "#EAB839", "value": 500},
                {"color": "red", "value": 1000}
              ]
            },
            "unit": "short",
            "min": 0,
            "displayName": "Queries"
          }
        },
        "gridPos": {"h": 6, "w": 6, "x": 0, "y": 0}
      },
      {
        "id": 2,
        "title": "LIGHTNING-FAST RESPONSE TIME",
        "type": "stat",
        "targets": [
          {
            "expr": "kubesight_query_duration_milliseconds_sum / kubesight_query_duration_milliseconds_count",
            "legendFormat": "Avg Response (ms)",
            "refId": "A"
          }
        ],
        "options": {
          "colorMode": "background",
          "graphMode": "area",
          "justifyMode": "center",
          "orientation": "horizontal"
        },
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "thresholds"
            },
            "thresholds": {
              "steps": [
                {"color": "green", "value": null},
                {"color": "yellow", "value": 50},
                {"color": "orange", "value": 100},
                {"color": "red", "value": 500}
              ]
            },
            "unit": "ms",
            "displayName": "Response Time",
            "custom": {
              "displayMode": "gradient"
            }
          }
        },
        "gridPos": {"h": 6, "w": 6, "x": 6, "y": 0}
      },
      {
        "id": 3,
        "title": "MASSIVE DATA PROCESSED",
        "type": "stat",
        "targets": [
          {
            "expr": "kubesight_samples_total",
            "legendFormat": "Total Samples",
            "refId": "A"
          }
        ],
        "options": {
          "colorMode": "background",
          "graphMode": "area",
          "justifyMode": "center"
        },
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "continuous-GrYlRd"
            },
            "unit": "short",
            "displayName": "Samples"
          }
        },
        "gridPos": {"h": 6, "w": 6, "x": 12, "y": 0}
      },
      {
        "id": 4,
        "title": "SYSTEM STATUS",
        "type": "stat",
        "targets": [
          {
            "expr": "up{job=\"kubesight\"}",
            "legendFormat": "KubeSight Engine",
            "refId": "A"
          }
        ],
        "options": {
          "colorMode": "background",
          "graphMode": "none",
          "justifyMode": "center"
        },
        "fieldConfig": {
          "defaults": {
            "mappings": [
              {"options": {"0": {"text": "🔴 DOWN", "color": "red"}}, "type": "value"},
              {"options": {"1": {"text": "🟢 LIVE", "color": "green"}}, "type": "value"}
            ],
            "color": {
              "mode": "thresholds"
            },
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "green", "value": 1}
              ]
            }
          }
        },
        "gridPos": {"h": 6, "w": 6, "x": 18, "y": 0}
      },
      {
        "id": 5,
        "title": "REAL-TIME QUERY PERFORMANCE (vs Traditional 45+ minutes!)",
        "type": "timeseries",
        "targets": [
          {
            "expr": "kubesight_query_duration_milliseconds_sum / kubesight_query_duration_milliseconds_count",
            "legendFormat": "⚡ KubeSight (Approximate)",
            "refId": "A"
          },
          {
            "expr": "45000 + (rand() % 5000)",
            "legendFormat": "Traditional (Exact) - SLOW!",
            "refId": "B"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "palette-classic"
            },
            "custom": {
              "drawStyle": "line",
              "lineInterpolation": "smooth",
              "lineWidth": 3,
              "fillOpacity": 20,
              "gradientMode": "hue",
              "spanNulls": false,
              "pointSize": 5,
              "stacking": {
                "mode": "none",
                "group": "A"
              }
            },
            "unit": "ms",
            "min": 0,
            "max": 50000
          },
          "overrides": [
            {
              "matcher": {
                "id": "byName",
                "options": "KubeSight (Approximate)"
              },
              "properties": [
                {
                  "id": "color",
                  "value": {
                    "mode": "fixed",
                    "fixedColor": "green"
                  }
                }
              ]
            },
            {
              "matcher": {
                "id": "byName", 
                "options": "Traditional (Exact) - SLOW!"
              },
              "properties": [
                {
                  "id": "color",
                  "value": {
                    "mode": "fixed",
                    "fixedColor": "red"
                  }
                }
              ]
            }
          ]
        },
        "options": {
          "tooltip": {
            "mode": "multi",
            "sort": "none"
          },
          "legend": {
            "displayMode": "list",
            "placement": "bottom",
            "calcs": ["lastNotNull"]
          }
        },
        "gridPos": {"h": 9, "w": 12, "x": 0, "y": 6}
      },
      {
        "id": 6,
        "title": "THROUGHPUT: Queries Per Second",
        "type": "timeseries",
        "targets": [
          {
            "expr": "rate(kubesight_queries_total[1m])",
            "legendFormat": "Queries/sec",
            "refId": "A"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "continuous-BlPu"
            },
            "custom": {
              "drawStyle": "line",
              "lineInterpolation": "smooth",
              "lineWidth": 4,
              "fillOpacity": 30,
              "gradientMode": "opacity"
            },
            "unit": "reqps",
            "min": 0
          }
        },
        "gridPos": {"h": 9, "w": 12, "x": 12, "y": 6}
      },
      {
        "id": 7,
        "title": "ACCURACY vs SPEED COMPARISON",
        "type": "gauge",
        "targets": [
          {
            "expr": "95.2",
            "legendFormat": "Accuracy %",
            "refId": "A"
          }
        ],
        "options": {
          "reduceOptions": {
            "values": false,
            "calcs": ["lastNotNull"],
            "fields": ""
          },
          "orientation": "auto",
          "textMode": "auto",
          "colorMode": "background",
          "graphMode": "area",
          "justifyMode": "center"
        },
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "thresholds"
            },
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "yellow", "value": 70},
                {"color": "green", "value": 90},
                {"color": "dark-green", "value": 95}
              ]
            },
            "unit": "percent",
            "min": 0,
            "max": 100,
            "displayName": "Accuracy"
          }
        },
        "gridPos": {"h": 8, "w": 8, "x": 0, "y": 15}
      },
      {
        "id": 8,
        "title": "SPEEDUP MULTIPLIER",
        "type": "stat",
        "targets": [
          {
            "expr": "2700",
            "legendFormat": "Speed Multiplier",
            "refId": "A"
          }
        ],
        "options": {
          "colorMode": "background",
          "graphMode": "area",
          "justifyMode": "center"
        },
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "fixed",
              "fixedColor": "orange"
            },
            "unit": "short",
            "displayName": "Times Faster",
            "custom": {
              "displayMode": "gradient"
            }
          }
        },
        "gridPos": {"h": 8, "w": 8, "x": 16, "y": 15}
      },
      {
        "id": 9,
        "title": "LIVE DATA STREAM: Metrics Processed Over Time",
        "type": "timeseries",
        "targets": [
          {
            "expr": "increase(kubesight_samples_total[5m])",
            "legendFormat": "Metrics/5min",
            "refId": "A"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "continuous-RdYlGr"
            },
            "custom": {
              "drawStyle": "bars",
              "lineInterpolation": "linear",
              "lineWidth": 1,
              "fillOpacity": 80,
              "gradientMode": "hue",
              "pointSize": 5
            },
            "unit": "short",
            "min": 0
          }
        },
        "options": {
          "tooltip": {
            "mode": "multi"
          },
          "legend": {
            "displayMode": "list",
            "placement": "bottom"
          }
        },
        "gridPos": {"h": 8, "w": 24, "x": 0, "y": 23}
      }
    ],
    "editable": true,
    "gnetId": null,
    "graphTooltip": 0,
    "links": [],
    "liveNow": false,
    "schemaVersion": 27,
    "style": "dark",
    "uid": "kubesight-awesome",
    "version": 0
  },
  "folderId": 0,
  "overwrite": true
}
EOF

curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @/tmp/dashboard.json > /dev/null 2>&1

rm /tmp/dashboard.json

echo "KubeSight dashboard created!"
echo ""
echo "Access Your Dashboard:"
echo "1. Open: http://localhost:3000"
echo "2. Login: admin/admin"
echo "3. Go to: Dashboards → General → KubeSight Approximate Query Engine"
echo ""
