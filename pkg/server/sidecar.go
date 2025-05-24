package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	pb "try/pkg/grpcapi"
)

type SidecarServer struct {
	pb.UnimplementedSidecarServiceServer
}

type BackendMetrics struct {
	CPUUsage       float64
	MemoryUsage    float64
	NetworkTraffic float64
}

var (
	backendHistory  = map[string][]BackendMetrics{"a": {}, "b": {}, "c": {}}
	requestCounts   = map[string]int{"a": 0, "b": 0, "c": 0}
	countLock       sync.Mutex
	maxHistory      = 10
	externalPortMap = map[string]string{
		"user-service-a": "x",
		"user-service-b": "y",
		"user-service-c": "z",
	}
	publicIP      = "x.y.z.w"
	prometheusURL = "http://x.y.z.w"
)

func getBackendMetrics(backend string) BackendMetrics {
	suffix := string(backend[len(backend)-1])
	podRegex := fmt.Sprintf("user-service-%s.*", suffix)

	cpu := queryPrometheusScalar(fmt.Sprintf(`rate(container_cpu_usage_seconds_total{pod=~"%s"}[5m])`, podRegex))
	mem := queryPrometheusScalar(fmt.Sprintf(`container_memory_usage_bytes{pod=~"%s"}`, podRegex)) / (1024 * 1024)
	netRx := queryPrometheusScalar(fmt.Sprintf(`rate(container_network_receive_bytes_total{pod=~"%s"}[5m])`, podRegex))
	netTx := queryPrometheusScalar(fmt.Sprintf(`rate(container_network_transmit_bytes_total{pod=~"%s"}[5m])`, podRegex))
	net := netRx + netTx

	cpuPercent := cpu * 100
	memPercent := (mem / 33560.0) * 100

	return BackendMetrics{cpuPercent, memPercent, net}
}

func queryPrometheusScalar(query string) float64 {
	url := fmt.Sprintf("%s/api/v1/query?query=%s", prometheusURL, query)
	url = strings.ReplaceAll(url, " ", "")
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to query Prometheus: %v", err)
		return 0
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result struct {
		Data struct {
			Result []struct {
				Value [2]interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Data.Result) == 0 {
		return 0
	}

	valueStr := result.Data.Result[0].Value[1].(string)
	var value float64
	fmt.Sscanf(valueStr, "%f", &value)
	return value
}

func updateHistory(backend string, metrics BackendMetrics) {
	history := backendHistory[backend]
	if len(history) >= maxHistory {
		history = history[1:]
	}
	backendHistory[backend] = append(history, metrics)
}

func combinedScore(current BackendMetrics, history []BackendMetrics) float64 {
	var avgCPU, avgMem, avgNet float64
	for _, m := range history {
		avgCPU += m.CPUUsage
		avgMem += m.MemoryUsage
		avgNet += m.NetworkTraffic
	}
	n := float64(len(history))
	if n > 0 {
		avgCPU /= n
		avgMem /= n
		avgNet /= n
	}
	blendedCPU := 0.7*avgCPU + 0.3*current.CPUUsage
	blendedMem := 0.7*avgMem + 0.3*current.MemoryUsage
	blendedNet := 0.7*avgNet + 0.3*current.NetworkTraffic
	return 0.5*blendedCPU + 0.3*blendedMem + 0.2*blendedNet
}

func selectBestBackend(serviceName string) (string, string) {
	backends := []string{"a", "b", "c"}
	scores := map[string]float64{}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Printf("\nPOD stats for %s\n", serviceName)
	fmt.Fprintln(w, "BACKEND\tCPU (%)\tMEMORY (%)\tNETWORK (B/s)")

	for _, suffix := range backends {
		backend := serviceName + "-" + suffix
		metrics := getBackendMetrics(backend)
		updateHistory(suffix, metrics)
		scores[suffix] = combinedScore(metrics, backendHistory[suffix])
		fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.2f\n", backend, metrics.CPUUsage, metrics.MemoryUsage, metrics.NetworkTraffic)
	}
	w.Flush()
	os.Stdout.Sync()
	fmt.Println("--------------------")

	best := backends[0]
	for _, b := range backends[1:] {
		if scores[b] < scores[best] {
			best = b
		}
	}

	selected := serviceName + "-" + best
	fmt.Printf("Selected: %s with score %.2f\n", selected, scores[best])
	return selected, best
}

func logRequestCount() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			countLock.Lock()
			fmt.Println("--- Request Count in Last 10s ---")
			for backend, count := range requestCounts {
				fmt.Printf("%s: %d requests\n", backend, count)
				requestCounts[backend] = 0
			}
			countLock.Unlock()
		}
	}()
}

func (s *SidecarServer) RouteRequest(ctx context.Context, req *pb.RouteRequestRequest) (*pb.RouteResponse, error) {
	if req == nil || req.ServiceName == "" {
		return nil, fmt.Errorf("invalid request: service name is empty")
	}

	selected, best := selectBestBackend(req.ServiceName)
	port := externalPortMap[selected]
	url := fmt.Sprintf("http://%s:%s", publicIP, port)

	start := time.Now()
	resp, err := http.Get(url)
	elapsed := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("error calling backend %s: %v", url, err)
	}
	defer resp.Body.Close()

	countLock.Lock()
	requestCounts[best]++
	countLock.Unlock()

	fmt.Printf("Response from %s: %s (took %v)\n\n", selected, resp.Status, elapsed)
	return &pb.RouteResponse{Backend: url}, nil
}

func init() {
	logRequestCount()
}
