package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Load struct {
	ID          string `json:"id"`
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Customer    string `json:"customer"`
	Carrier     string `json:"carrier"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type TurvoAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type TurvoShipment struct {
	ID      string `json:"id"`
	Details struct {
		Contributors []struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"contributors"`
		Stops []struct {
			Type     string `json:"type"`
			Location struct {
				City  string `json:"city"`
				State string `json:"state"`
			} `json:"location"`
		} `json:"stops"`
	} `json:"details"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type TurvoShipmentsResponse struct {
	Shipments []TurvoShipment `json:"shipments"`
}

func getTurvoToken() (string, error) {
	payload := map[string]string{
		"username":   os.Getenv("TURVO_USERNAME"),
		"password":   os.Getenv("TURVO_PASSWORD"),
		"grant_type": "password",
	}
	jsonData, _ := json.Marshal(payload)

	authURL := fmt.Sprintf("%s/v1/oauth/token?client_id=%s&client_secret=%s",
		os.Getenv("TURVO_AUTH_URL"),
		os.Getenv("TURVO_CLIENT_ID"),
		os.Getenv("TURVO_CLIENT_SECRET"))

	req, _ := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", os.Getenv("TURVO_API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp TurvoAuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)
	return authResp.AccessToken, nil
}

// Handler is the main entry point for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	switch r.Method {
	case "GET":
		handleGetLoads(w, r)
	case "POST":
		handleCreateLoad(w, r)
	case "DELETE":
		handleDeleteLoad(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetLoads(w http.ResponseWriter, r *http.Request) {
	token, err := getTurvoToken()
	if err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	req, _ := http.NewRequest("GET", "https://my-sandbox.turvo.com/api/shipments/list", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch shipments", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var turvoResp TurvoShipmentsResponse
	json.NewDecoder(resp.Body).Decode(&turvoResp)

	var loads []Load
	for _, shipment := range turvoResp.Shipments {
		load := Load{
			ID:        shipment.ID,
			Status:    shipment.Status,
			CreatedAt: shipment.CreatedAt,
		}

		// Extract origin and destination
		for _, stop := range shipment.Details.Stops {
			location := fmt.Sprintf("%s, %s", stop.Location.City, stop.Location.State)
			if stop.Type == "pickup" {
				load.Origin = location
			} else if stop.Type == "delivery" {
				load.Destination = location
			}
		}

		// Extract customer and carrier
		for _, contributor := range shipment.Details.Contributors {
			if contributor.Type == "customer" {
				load.Customer = contributor.Name
			} else if contributor.Type == "carrier" {
				load.Carrier = contributor.Name
			}
		}

		loads = append(loads, load)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loads)
}

func handleCreateLoad(w http.ResponseWriter, r *http.Request) {
	var load Load
	json.NewDecoder(r.Body).Decode(&load)

	token, err := getTurvoToken()
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	shipmentData := map[string]interface{}{
		"details": map[string]interface{}{
			"stops": []map[string]interface{}{
				{
					"type": "pickup",
					"location": map[string]string{
						"city":  strings.Split(load.Origin, ",")[0],
						"state": strings.TrimSpace(strings.Split(load.Origin, ",")[1]),
					},
				},
				{
					"type": "delivery",
					"location": map[string]string{
						"city":  strings.Split(load.Destination, ",")[0],
						"state": strings.TrimSpace(strings.Split(load.Destination, ",")[1]),
					},
				},
			},
			"contributors": []map[string]string{
				{"type": "customer", "name": load.Customer},
				{"type": "carrier", "name": load.Carrier},
			},
		},
		"status":    "active",
		"createdAt": time.Now().Format(time.RFC3339),
	}

	jsonData, _ := json.Marshal(shipmentData)
	req, _ := http.NewRequest("POST", "https://my-sandbox.turvo.com/api/shipments", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to create shipment", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var createdShipment map[string]interface{}
	json.Unmarshal(body, &createdShipment)

	if id, ok := createdShipment["id"].(string); ok {
		load.ID = id
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(load)
}

func handleDeleteLoad(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter required", http.StatusBadRequest)
		return
	}

	token, err := getTurvoToken()
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	req, _ := http.NewRequest("DELETE", "https://my-sandbox.turvo.com/api/shipments/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to delete shipment", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Load deleted successfully"}`))
}
