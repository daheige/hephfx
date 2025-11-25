package monitor

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/daheige/hephfx/monitor/gpprof"
)

// InitMonitor 初始化prometheus和go pprof
// 添加prometheus性能监控指标
// 假设port 为 2337 那么访问地址如下：
// 访问地址：http://localhost:2337/metrics
// 访问地址：http://localhost:2337/debug/pprof/
func InitMonitor(port uint16, isWeb ...bool) {
	if len(isWeb) > 0 && isWeb[0] {
		prometheus.MustRegister(WebRequestTotal)
		prometheus.MustRegister(WebRequestDuration)
	}

	prometheus.MustRegister(CpuTemp)
	prometheus.MustRegister(HdFailures)

	// 性能监控的端口port+1000,只能在内网访问
	httpMux := gpprof.New()

	// 添加prometheus metrics处理器
	log.Printf("Starting monitor metrics http server on:%d request path is /metrics", port)
	httpMux.Handle("/metrics", promhttp.Handler())
	gpprof.Run(httpMux, port)
}
