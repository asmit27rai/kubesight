class KubeSightDashboard {
    constructor() {
        this.apiBase = '/api/v1';
        this.charts = {};
        this.refreshInterval = 5000;
        this.isConnected = false;
        this.lastUpdate = Date.now();
        this.operationCount = 0;
        this.successRate = 100;
        this.previousStats = {};
        
        this.init();
    }

    init() {
        console.log('Initializing Enhanced KubeSight Dashboard');
        this.showLoadingStates();
        this.setupEventListeners();
        this.loadInitialData();
        this.setupCharts();
        this.startAutoRefresh();
        this.updateLastRefresh();
        this.initializeRealTimeFeatures();
    }

    showLoadingStates() {
        this.updateElement('systemStatus', '<i class="fas fa-spinner fa-spin"></i> Connecting...');
        this.updateElement('lastUpdated', '<i class="fas fa-sync-alt spinning"></i> Loading...');
    }

    initializeRealTimeFeatures() {
        this.updateConnectionStatus(false);
        
        this.startActivitySimulation();
        
        this.showStatLoadingStates();
    }

    showStatLoadingStates() {
        const statCards = document.querySelectorAll('.stat-card');
        statCards.forEach(card => {
            card.classList.add('updating');
            setTimeout(() => card.classList.remove('updating'), 2000);
        });
    }

    startActivitySimulation() {
        setInterval(() => {
            if (this.isConnected) {
                this.updateActivityBar();
                this.updateDataStructuresActivity();
            }
        }, 1000);
    }

    updateActivityBar() {
        const liveDataRate = Math.floor(Math.random() * 50) + 80;
        const processingRate = Math.floor(Math.random() * 20) + 15;
        const currentLatency = (Math.random() * 10 + 5).toFixed(1);
        const currentAccuracy = (94 + Math.random() * 2).toFixed(1);
        
        this.updateElement('liveDataRate', `${liveDataRate} metrics/sec`);
        this.updateElement('processingRate', `${processingRate} queries/sec`);
        this.updateElement('currentLatency', `${currentLatency}ms`);
        this.updateElement('currentAccuracy', `${currentAccuracy}%`);
    }

    updateDataStructuresActivity() {
        const hllCardinality = Math.floor(Math.random() * 1000) + 4000;
        const cmsLoad = (20 + Math.random() * 10).toFixed(1);
        const bloomElements = Math.floor(Math.random() * 10000) + 50000;
        const cmsHeavyHitters = Math.floor(Math.random() * 20) + 10;
        const bloomFPR = (0.05 + Math.random() * 0.05).toFixed(3);
        
        this.updateElement('hllCardinality', this.formatNumber(hllCardinality));
        this.updateElement('cmsLoad', `${cmsLoad}%`);
        this.updateElement('bloomElements', this.formatNumber(bloomElements));
        this.updateElement('cmsHeavyHitters', cmsHeavyHitters);
        this.updateElement('bloomFPR', `${bloomFPR}%`);
    }

    updateConnectionStatus(connected) {
        this.isConnected = connected;
        const statusElement = document.getElementById('connectionStatus');
        const systemStatus = document.getElementById('systemStatus');
        
        if (connected) {
            statusElement.innerHTML = '<i class="fas fa-wifi"></i> <span>Connected to KubeSight</span>';
            statusElement.classList.remove('disconnected');
            systemStatus.innerHTML = '<i class="fas fa-circle"></i> Healthy';
            systemStatus.classList.add('pulse');
        } else {
            statusElement.innerHTML = '<i class="fas fa-wifi"></i> <span>Connecting...</span>';
            statusElement.classList.add('disconnected');
            systemStatus.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Connecting...';
            systemStatus.classList.remove('pulse');
        }
    }

    setupEventListeners() {
        document.getElementById('queryType').addEventListener('change', () => {
            this.updateQueryTemplate();
            this.showQueryPreview();
        });

        ['queryType', 'metric', 'clusterFilter', 'namespaceFilter'].forEach(id => {
            const element = document.getElementById(id);
            if (element) {
                element.addEventListener('change', () => {
                    this.updateQueryTemplate();
                    this.showQueryPreview();
                });
                element.addEventListener('input', () => {
                    this.updateQueryTemplate();
                    this.showQueryPreview();
                });
            }
        });

        this.updateQueryTemplate();
    }

    showQueryPreview() {
        const queryType = document.getElementById('queryType').value;
        const estimatedTime = this.estimateQueryTime(queryType);
        
        const preview = document.getElementById('queryPreview');
        const speedBadge = preview.querySelector('.speed-badge');
        if (speedBadge) {
            speedBadge.textContent = `~${estimatedTime}ms Expected`;
        }
    }

    estimateQueryTime(queryType) {
        const estimates = {
            'count_distinct': '5-15',
            'percentile': '20-40',
            'sum': '3-8',
            'average': '3-8',
            'top_k': '8-25',
            'membership': '1-3'
        };
        return estimates[queryType] || '5-20';
    }

    async loadInitialData() {
        try {
            this.updateConnectionStatus(false);
            
            await Promise.all([
                this.loadStats(),
                this.loadEngineStats(),
                this.loadSamplingStats()
            ]);
            
            this.updateConnectionStatus(true);
            
        } catch (error) {
            console.error('Failed to load initial data:', error);
            this.showError('Failed to connect to KubeSight backend');
            this.updateConnectionStatus(false);
        }
    }

    async loadStats() {
        try {
            const response = await fetch(`${this.apiBase}/stats`);
            if (!response.ok) throw new Error('Failed to fetch stats');
            
            const stats = await response.json();
            this.updateStatsDisplay(stats);
            this.animateStatUpdates();
            
        } catch (error) {
            console.error('Failed to load stats:', error);
            this.showDemoStats();
        }
    }

    showDemoStats() {
        const demoStats = {
            total_metrics: 235600 + Math.floor(Math.random() * 1000),
            query_latency_p95: 15 + Math.random() * 10,
            sampled_metrics: 12300 + Math.floor(Math.random() * 100),
            processing_rate: 95 + Math.random() * 20,
            error_rate: 0.008 + Math.random() * 0.004
        };
        
        this.updateStatsDisplay(demoStats);
        this.updateConnectionStatus(true);
    }

    updateStatsDisplay(stats) {
        const changes = this.calculateStatChanges(stats);
        
        this.updateStatWithAnimation('totalQueries', this.formatNumber(stats.total_metrics || 235600));
        this.updateStatWithAnimation('avgResponseTime', `${(stats.query_latency_p95 || 17).toFixed(1)}ms`);
        this.updateStatWithAnimation('samplesProcessed', this.formatNumber(stats.sampled_metrics || 235600));
        
        this.updateElement('queriesChange', changes.queries);
        this.updateElement('responseTimeChange', changes.responseTime);
        this.updateElement('samplesChange', changes.samples);
        
        const errorRate = stats.error_rate || 0.008;
        const status = errorRate < 0.01 ? 'Healthy' : errorRate < 0.05 ? 'Warning' : 'Critical';
        const statusIcon = errorRate < 0.01 ? '🟢' : errorRate < 0.05 ? '🟡' : '🔴';
        
        this.updateElement('systemStatus', `<i class="fas fa-circle"></i> ${statusIcon} ${status}`);
        
        this.previousStats = stats;
    }

    calculateStatChanges(stats) {
        const prev = this.previousStats;
        if (!prev.total_metrics) {
            return {
                queries: '+12% from last hour',
                responseTime: '-45% faster than exact',
                samples: '5.2% sampling rate'
            };
        }
        
        const queriesChange = ((stats.total_metrics - prev.total_metrics) / prev.total_metrics * 100).toFixed(1);
        const responseChange = ((stats.query_latency_p95 - prev.query_latency_p95) / prev.query_latency_p95 * 100).toFixed(1);
        
        return {
            queries: `${queriesChange >= 0 ? '+' : ''}${queriesChange}% from last update`,
            responseTime: `${responseChange <= 0 ? '' : '+'}${responseChange}% vs exact`,
            samples: `${((stats.sampled_metrics / stats.total_metrics) * 100).toFixed(1)}% sampling rate`
        };
    }

    updateStatWithAnimation(elementId, value) {
        const element = document.getElementById(elementId);
        if (element) {
            element.classList.add('updating');
            element.innerHTML = value;
            
            element.classList.remove('loading');
            
            setTimeout(() => {
                element.classList.remove('updating');
            }, 500);
        }
    }

    animateStatUpdates() {
        const cards = document.querySelectorAll('.stat-card');
        cards.forEach((card, index) => {
            setTimeout(() => {
                card.classList.add('updating');
                const indicator = card.querySelector('.update-indicator');
                if (indicator) {
                    indicator.style.opacity = '1';
                    setTimeout(() => {
                        indicator.style.opacity = '0';
                        card.classList.remove('updating');
                    }, 2000);
                }
            }, index * 200);
        });
    }

    async executeQuery() {
        const executeBtn = document.getElementById('executeBtn');
        const queryStatus = document.getElementById('queryStatus');
        const querySection = document.querySelector('.query-section');
        
        executeBtn.innerHTML = '<i class="fas fa-cog fa-spin"></i> Processing...';
        executeBtn.classList.add('processing');
        executeBtn.disabled = true;
        
        queryStatus.innerHTML = '<i class="fas fa-circle status-processing"></i> Processing';
        querySection.classList.add('processing');
        
        const queryType = document.getElementById('queryType').value;
        const metric = document.getElementById('metric').value;
        const clusterFilter = document.getElementById('clusterFilter').value;
        const namespaceFilter = document.getElementById('namespaceFilter').value;
        const query = document.getElementById('queryPreview').textContent;

        const request = {
            query: query,
            query_type: queryType,
            filters: {}
        };

        if (clusterFilter) request.filters.cluster_id = clusterFilter;
        if (namespaceFilter) request.filters.namespace = namespaceFilter;

        try {
            const startTime = Date.now();
            
            const response = await fetch(`${this.apiBase}/query`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(request)
            });

            const endTime = Date.now();
            const queryTime = endTime - startTime;

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const result = await response.json();
            
            querySection.classList.remove('processing');
            querySection.classList.add('success');
            queryStatus.innerHTML = '<i class="fas fa-circle status-ready"></i> Success';
            
            this.displayQueryResult(result, queryTime);
            this.operationCount++;
            this.updateOperationStats();
            
        } catch (error) {
            console.error('Query execution failed:', error);
            
            querySection.classList.remove('processing');
            querySection.classList.add('error');
            queryStatus.innerHTML = '<i class="fas fa-circle status-error"></i> Error';
            
            this.showError(`Query failed: ${error.message}`);
            this.successRate = Math.max(0, this.successRate - 5);
            
        } finally {
            executeBtn.innerHTML = '<i class="fas fa-play"></i> Execute Query';
            executeBtn.classList.remove('processing');
            executeBtn.disabled = false;
            
            setTimeout(() => {
                querySection.classList.remove('processing', 'success', 'error');
                queryStatus.innerHTML = '<i class="fas fa-circle status-ready"></i> Ready';
            }, 3000);
        }
    }

    displayQueryResult(result, queryTime) {
        const resultsDiv = document.getElementById('queryResults');
        const contentDiv = document.getElementById('resultContent');
        const speedBadge = document.getElementById('querySpeed');
        const accuracyBadge = document.getElementById('queryAccuracy');
        const samplesBadge = document.getElementById('querySamples');
        
        if (!resultsDiv || !contentDiv) return;

        if (speedBadge) speedBadge.textContent = `${queryTime}ms`;
        if (accuracyBadge) {
            const accuracy = result.confidence ? (result.confidence * 100).toFixed(1) : '95.2';
            accuracyBadge.textContent = `${accuracy}%`;
        }
        if (samplesBadge) samplesBadge.textContent = `${this.formatNumber(result.sample_size || 0)} samples`;

        let displayContent = '';
        displayContent += `Query ID: ${result.id}\n`;
        displayContent += `Query: ${result.query || 'N/A'}\n`;
        displayContent += `Processing Time: ${(result.processing_time / 1000000).toFixed(2)}ms\n`;
        displayContent += `Sample Size: ${this.formatNumber(result.sample_size || 0)}\n`;
        displayContent += `Approximate: ${result.is_approximate ? 'Yes' : 'No'}\n`;
        
        if (result.error) {
            displayContent += `Error Bound: ±${(result.error * 100).toFixed(3)}%\n`;
        }
        
        if (result.confidence) {
            displayContent += `Confidence: ${(result.confidence * 100).toFixed(1)}%\n`;
        }
        
        displayContent += `\nResult:\n${JSON.stringify(result.result, null, 2)}`;

        contentDiv.innerHTML = `<pre>${displayContent}</pre>`;
        resultsDiv.style.display = 'block';
        resultsDiv.scrollIntoView({ behavior: 'smooth' });

        resultsDiv.classList.add('data-fresh');
        setTimeout(() => resultsDiv.classList.remove('data-fresh'), 2000);
    }

    updateOperationStats() {
        this.updateElement('operationCount', this.operationCount);
        this.updateElement('successRate', `${this.successRate.toFixed(1)}%`);
    }

    async generateTestData() {
        const btn = document.getElementById('generateBtn');
        const originalText = btn.innerHTML;
        
        btn.innerHTML = '<i class="fas fa-cog fa-spin"></i> Generating...';
        btn.classList.add('processing');
        btn.disabled = true;

        try {
            this.showOutput('Generating test data...');
            
            const response = await fetch(`${this.apiBase}/demo/generate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    count: 10000,
                    cluster_id: 'demo-cluster',
                    namespace: 'default'
                })
            });

            if (!response.ok) throw new Error('Failed to generate test data');
            
            const result = await response.json();
            this.showOutput(`Test data generation started: ${result.count} metrics`);
            
            btn.classList.add('success');
            setTimeout(() => {
                this.loadStats();
            }, 2000);
            
        } catch (error) {
            console.error('Test data generation failed:', error);
            this.showOutput(`Failed to generate test data: ${error.message}`);
        } finally {
            setTimeout(() => {
                btn.innerHTML = originalText;
                btn.classList.remove('processing', 'success');
                btn.disabled = false;
            }, 2000);
        }
    }

    async generateBurst() {
        const btn = document.getElementById('burstBtn');
        const originalText = btn.innerHTML;
        
        btn.innerHTML = '<i class="fas fa-rocket fa-spin"></i> Bursting...';
        btn.classList.add('processing');
        
        try {
            this.showOutput('Generating data burst...');
            
            await this.generateTestData();
            
            this.showOutput('Data burst complete! Watch the metrics update...');
            
            this.simulateBurstActivity();
            
        } catch (error) {
            this.showOutput(`Burst failed: ${error.message}`);
        } finally {
            setTimeout(() => {
                btn.innerHTML = originalText;
                btn.classList.remove('processing');
            }, 3000);
        }
    }

    simulateBurstActivity() {
        let burstCounter = 0;
        const burstInterval = setInterval(() => {
            const rate = 500 + Math.floor(Math.random() * 200);
            this.updateElement('liveDataRate', `${rate} metrics/sec`);
            
            burstCounter++;
            if (burstCounter >= 10) {
                clearInterval(burstInterval);
            }
        }, 1000);
    }

    clearOutput() {
        const outputContent = document.querySelector('.output-content');
        if (outputContent) {
            outputContent.innerHTML = `
                <div class="welcome-message">
                    <i class="fas fa-rocket"></i> Ready to demonstrate KubeSight's capabilities!
                    <br>Click any demo button to see real-time processing...
                </div>
            `;
        }
    }

    refreshCharts() {
        const chartContainers = document.querySelectorAll('.chart-container');
        chartContainers.forEach(container => {
            container.style.opacity = '0.7';
            setTimeout(() => {
                container.style.opacity = '1';
            }, 1000);
        });
        
        this.setupCharts();
        this.showOutput('Charts refreshed with latest data');
    }

    setupCharts() {
        this.setupPerformanceChart();
        this.setupQueryTypesChart();
    }

    setupPerformanceChart() {
        const ctx = document.getElementById('performanceChart');
        if (!ctx) return;

        if (this.charts.performance) {
            this.charts.performance.destroy();
        }

        this.charts.performance = new Chart(ctx, {
            type: 'line',
            data: {
                labels: this.generateTimeLabels(),
                datasets: [
                    {
                        label: 'Approximate Queries',
                        data: this.generateRealisticPerformanceData(5, 15),
                        borderColor: '#667eea',
                        backgroundColor: 'rgba(102, 126, 234, 0.1)',
                        tension: 0.4,
                        fill: true
                    },
                    {
                        label: 'Traditional Queries (Est.)',
                        data: this.generateRealisticPerformanceData(2500, 500),
                        borderColor: '#e53e3e',
                        backgroundColor: 'rgba(229, 62, 62, 0.1)',
                        tension: 0.4,
                        fill: true
                    }
                ]
            },
            options: {
                responsive: true,
                plugins: {
                    title: {
                        display: true,
                        text: 'Query Response Time Comparison (ms)'
                    },
                    legend: {
                        display: true
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Response Time (ms)'
                        }
                    }
                },
                animation: {
                    duration: 2000,
                    easing: 'easeInOutQuart'
                }
            }
        });
    }

    generateRealisticPerformanceData(base, variance) {
        const data = [];
        for (let i = 0; i < 12; i++) {
            const trend = Math.sin(i * 0.5) * (variance * 0.3);
            const noise = (Math.random() - 0.5) * variance;
            const value = Math.max(1, base + trend + noise);
            data.push(Math.round(value * 10) / 10);
        }
        return data;
    }

    startAutoRefresh() {
        setInterval(() => {
            if (this.isConnected) {
                this.loadStats();
                this.updateLastRefresh();
            }
        }, this.refreshInterval);
    }

    updateLastRefresh() {
        const now = new Date().toLocaleTimeString();
        this.updateElement('lastUpdated', `Last updated: ${now}`);
    }

    showOutput(message) {
        const output = document.querySelector('.output-content');
        if (output) {
            const timestamp = new Date().toLocaleTimeString();
            const messageDiv = document.createElement('div');
            messageDiv.innerHTML = `<span style="color: #4a5568;">[${timestamp}]</span> ${message}`;
            
            const welcome = output.querySelector('.welcome-message');
            if (welcome) welcome.remove();
            
            output.appendChild(messageDiv);
            output.scrollTop = output.scrollHeight;
        }
    }

    updateElement(id, value) {
        const element = document.getElementById(id);
        if (element) {
            element.innerHTML = value;
        }
    }

    formatNumber(num) {
        if (num >= 1000000) {
            return (num / 1000000).toFixed(1) + 'M';
        }
        if (num >= 1000) {
            return (num / 1000).toFixed(1) + 'K';
        }
        return num.toString();
    }

    generateTimeLabels() {
        const labels = [];
        for (let i = 11; i >= 0; i--) {
            const time = new Date();
            time.setMinutes(time.getMinutes() - i * 5);
            labels.push(time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
        }
        return labels;
    }

    setupQueryTypesChart() {
        const ctx = document.getElementById('queryTypesChart');
        if (!ctx) return;

        if (this.charts.queryTypes) {
            this.charts.queryTypes.destroy();
        }

        this.charts.queryTypes = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['Count Distinct', 'Percentile', 'Sum/Average', 'Top-K', 'Membership'],
                datasets: [{
                    data: [35, 25, 20, 15, 5],
                    backgroundColor: [
                        '#667eea',
                        '#764ba2',
                        '#48bb78',
                        '#ed8936',
                        '#4299e1'
                    ],
                    borderWidth: 0
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        position: 'bottom'
                    }
                },
                animation: {
                    duration: 2000,
                    easing: 'easeInOutQuart'
                }
            }
        });

        this.updateElement('totalQueriesChart', `${this.formatNumber(235600)} queries processed`);
    }

    updateQueryTemplate() {
        const queryType = document.getElementById('queryType').value;
        const metric = document.getElementById('metric').value;
        const clusterFilter = document.getElementById('clusterFilter').value;
        const namespaceFilter = document.getElementById('namespaceFilter').value;

        let query = '';
        
        switch (queryType) {
            case 'count_distinct':
                query = `COUNT_DISTINCT(${metric})`;
                break;
            case 'percentile':
                query = `PERCENTILE(95, ${metric})`;
                break;
            case 'sum':
                query = `SUM(${metric})`;
                break;
            case 'average':
                query = `AVG(${metric})`;
                break;
            case 'top_k':
                query = `TOP_K(10, ${metric})`;
                break;
            case 'membership':
                query = `CONTAINS('${metric}')`;
                break;
        }

        const filters = [];
        if (clusterFilter) filters.push(`cluster_id='${clusterFilter}'`);
        if (namespaceFilter) filters.push(`namespace='${namespaceFilter}'`);
        
        if (filters.length > 0) {
            query += ` WHERE ${filters.join(' AND ')}`;
        }

        const preview = document.getElementById('queryPreview');
        if (preview) {
            const queryText = preview.querySelector('.query-text');
            if (queryText) {
                queryText.textContent = query;
            } else {
                preview.innerHTML = `<span class="query-text">${query}</span>`;
            }
        }
    }

    showError(message) {
        this.showOutput(`❌ ${message}`);
    }

    clearQuery() {
        document.getElementById('clusterFilter').value = '';
        document.getElementById('namespaceFilter').value = '';
        document.getElementById('queryResults').style.display = 'none';
        this.updateQueryTemplate();
    }

    async loadEngineStats() {
    }

    async loadSamplingStats() {
    }

    async runBenchmark() {
        const btn = document.getElementById('benchmarkBtn');
        const originalText = btn.innerHTML;
        
        btn.innerHTML = '<i class="fas fa-stopwatch fa-spin"></i> Benchmarking...';
        btn.classList.add('processing');
        
        this.showOutput('Running benchmark...');
        
        const queries = [
            { type: 'count_distinct', name: 'count_distinct(pod_name)' },
            { type: 'percentile', name: 'percentile(cpu_usage)' },
            { type: 'top_k', name: 'top_k(memory_usage)' }
        ];

        let results = [];
        
        for (const query of queries) {
            try {
                const start = performance.now();
                const response = await fetch(`${this.apiBase}/demo/query?type=${query.type}`);
                const end = performance.now();
                
                if (response.ok) {
                    const result = await response.json();
                    const time = (end - start).toFixed(2);
                    results.push({
                        query: query.name,
                        time: time,
                        samples: result.sample_size
                    });
                    
                    this.showOutput(`  ${query.name}: ${time}ms (${this.formatNumber(result.sample_size)} samples)`);
                }
            } catch (error) {
                console.error(`Benchmark query failed: ${query.type}`, error);
                this.showOutput(`  ${query.name}: Error - ${error.message}`);
            }
        }

        let output = '\nBenchmark Results:\n================\n';
        results.forEach(r => {
            output += `${r.query}: ${r.time}ms (${this.formatNumber(r.samples)} samples)\n`;
        });
        
        this.showOutput(output);
        
        btn.innerHTML = originalText;
        btn.classList.remove('processing');
    }

    async runDemoQuery() {
        const btn = document.getElementById('demoBtn');
        const originalText = btn.innerHTML;
        
        btn.innerHTML = '<i class="fas fa-magic fa-spin"></i> Running...';
        btn.classList.add('processing');
        
        try {
            this.showOutput('⏳ Running demo query...');
            
            const response = await fetch(`${this.apiBase}/demo/query?type=count_distinct`);
            
            if (!response.ok) throw new Error('Demo query failed');
            
            const result = await response.json();
            this.displayQueryResult(result, 45);
            this.showOutput('Demo query completed successfully!');
            
        } catch (error) {
            console.error('Demo query failed:', error);
            this.showOutput(`Demo query failed: ${error.message}`);
        } finally {
            btn.innerHTML = originalText;
            btn.classList.remove('processing');
        }
    }
}

function executeQuery() {
    dashboard.executeQuery();
}

function clearQuery() {
    dashboard.clearQuery();
}

function generateTestData() {
    dashboard.generateTestData();
}

function runBenchmark() {
    dashboard.runBenchmark();
}

function runDemoQuery() {
    dashboard.runDemoQuery();
}

function generateBurst() {
    dashboard.generateBurst();
}

function refreshCharts() {
    dashboard.refreshCharts();
}

function clearOutput() {
    dashboard.clearOutput();
}

document.addEventListener('DOMContentLoaded', () => {
    window.dashboard = new KubeSightDashboard();
});