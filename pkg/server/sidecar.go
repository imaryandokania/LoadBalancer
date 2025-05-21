package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
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

type MLModel struct {
	Weights   []float64 `json:"weights"`
	Intercept float64   `json:"intercept"`
}

var model MLModel
var backendHistory = map[string][]BackendMetrics{
	"a": {},
	"b": {},
	"c": {},
}

const maxHistory = 10

func loadModel(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to load model: %v", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&model); err != nil {
		log.Fatalf("Failed to decode model: %v", err)
	}
	fmt.Printf("Loaded ML model: weights=%v, intercept=%.2f\n", model.Weights, model.Intercept)
}

// Predict using loaded model
func mlPredict(m BackendMetrics) float64 {
	x := []float64{m.CPUUsage, m.MemoryUsage, m.NetworkTraffic}
	score := model.Intercept
	for i := range model.Weights {
		score += model.Weights[i] * x[i]
	}
	return score
}

func getBackendMetrics(backend string) BackendMetrics {
	cpu := rand.Float64() * 100
	mem := rand.Float64() * 100
	net := rand.Float64() * 200
	fmt.Printf("%s: CPU=%.2f, Memory=%.2f, Network=%.2f\n", backend, cpu, mem, net)
	return BackendMetrics{
		CPUUsage:       cpu,
		MemoryUsage:    mem,
		NetworkTraffic: net,
	}
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
	count := float64(len(history))
	if count > 0 {
		avgCPU /= count
		avgMem /= count
		avgNet /= count
	}

	// Weighted sum of past trend and current values
	score := 0.7*mlPredict(BackendMetrics{avgCPU, avgMem, avgNet}) +
		0.3*mlPredict(current)
	return score
}

func selectBestBackend(serviceName string) string {
	backends := []string{"a", "b", "c"}
	scores := map[string]float64{}
	fmt.Printf("POD stats for %s\n", serviceName)
	for _, suffix := range backends {
		backend := serviceName + "-" + suffix + ":8080"
		metrics := getBackendMetrics(backend)
		updateHistory(suffix, metrics)
		score := combinedScore(metrics, backendHistory[suffix])
		scores[suffix] = score
	}

	best := backends[0]
	for _, b := range backends[1:] {
		if scores[b] < scores[best] {
			best = b
		}
	}

	selected := serviceName + "-" + best + ":8080"
	fmt.Printf("ML trend-aware selected: %s with score %.2f\n", selected, scores[best])
	return selected
}

func (s *SidecarServer) RouteRequest(ctx context.Context, req *pb.RouteRequestRequest) (*pb.RouteResponse, error) {
	fmt.Printf("Routing request to service: %s\n", req.ServiceName)
	selected := selectBestBackend(req.ServiceName)
	return &pb.RouteResponse{Backend: "http://" + selected}, nil
}

func init() {
	loadModel("model.json")
}
