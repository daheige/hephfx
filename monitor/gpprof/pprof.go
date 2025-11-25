package gpprof

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"time"
)

// New 创建一个http ServeMux实例
func New() *http.ServeMux {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/debug/pprof/", pprof.Index)
	httpMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	httpMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	httpMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	httpMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	httpMux.HandleFunc("/check", Check)

	return httpMux
}

// Run 运行PProf性能监控服务
// 性能监控的端口port只能在内网访问
// 一般在程序启动init或main函数中执行
// httpMux 表示*http.ServeMux
// port表示pprof运行的端口
func Run(httpMux *http.ServeMux, port uint16) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("PProf exec recover: ", err)
			}
		}()

		address := fmt.Sprintf("0.0.0.0:%d", port)
		log.Printf("Starting go PProf run on::%d request path is /debug/pprof\n", port)
		if err := http.ListenAndServe(address, httpMux); err != nil {
			log.Println("PProf listen error:", err)
		}
	}()
}

// Check PProf心跳检测
func Check(w http.ResponseWriter, r *http.Request) {
	m := map[string]interface{}{
		"alive": true,
		"time":  time.Now().Format("2006-01-02 15:04:05"),
	}

	b, _ := json.Marshal(m)
	_, _ = w.Write(b)
}
