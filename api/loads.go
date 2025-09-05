package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Load struct {
	ID          string `json:"id"`
	NumericID   string `json:"numericId,omitempty"`
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
	ID            interface{} `json:"id"`
	ProjectFields struct {
		Title struct {
			DisplayID string `json:"displayId"`
		} `json:"title"`
		ShipmentID int `json:"shipmentId"`
	} `json:"projectFields"`
	Details struct {
		Lane struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"lane"`
		CustomerOrders []struct {
			Customer struct {
				Name string `json:"name"`
			} `json:"customer"`
		} `json:"customer_orders"`
		Contributors []struct {
			ContributorUser struct {
				Name string `json:"name"`
			} `json:"contributorUser"`
			Title struct {
				Value string `json:"value"`
			} `json:"title"`
		} `json:"contributors"`
		Status struct {
			Description string `json:"description"`
		} `json:"status"`
		Date string `json:"date"`
	} `json:"details"`
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
			ID:          shipment.ProjectFields.Title.DisplayID,
			NumericID:   fmt.Sprintf("%d", shipment.ProjectFields.ShipmentID),
			Origin:      shipment.Details.Lane.Start,
			Destination: shipment.Details.Lane.End,
			Status:      shipment.Details.Status.Description,
			CreatedAt:   shipment.Details.Date,
		}

		// Extract customer name
		if len(shipment.Details.CustomerOrders) > 0 {
			load.Customer = shipment.Details.CustomerOrders[0].Customer.Name
		}

		// Extract carrier/broker from contributors
		for _, contributor := range shipment.Details.Contributors {
			if contributor.Title.Value == "Broker" {
				load.Carrier = contributor.ContributorUser.Name
				break
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
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	payload := map[string]interface{}{
		"ltlShipment": false,
		"startDate": map[string]interface{}{
			"date":     time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"timeZone": "America/New_York",
		},
		"endDate": map[string]interface{}{
			"date":     time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			"timeZone": "America/New_York",
		},
		"lane": map[string]interface{}{
			"start": load.Origin,
			"end":   load.Destination,
		},
		"customerOrder": []map[string]interface{}{
			{
				"customer": map[string]interface{}{
					"name": load.Customer,
					"id":   load.Customer,
				},
				"items": []map[string]interface{}{
					{
						"name":        "General Freight",
						"description": "General freight",
						"qty":         1,
						"quantity":    1,
						"unit": map[string]interface{}{
							"key":   "6000",
							"value": "Pieces",
						},
						"itemCategory": map[string]interface{}{
							"key":   "22300",
							"value": "Other",
						},
					},
				},
			},
		},
		"carrierOrder": []map[string]interface{}{},
	}

	jsonData, _ := json.Marshal(payload)
	turvoReq, _ := http.NewRequest("POST", "https://my-sandbox.turvo.com/api/shipments", bytes.NewBuffer(jsonData))
	turvoReq.Header.Set("Authorization", "Bearer "+token)
	turvoReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(turvoReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create shipment: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		http.Error(w, fmt.Sprintf("Turvo API error (status %d): %s", resp.StatusCode, string(body)), resp.StatusCode)
		return
	}

	var createdShipment map[string]interface{}
	json.Unmarshal(body, &createdShipment)

	if id, ok := createdShipment["id"]; ok {
		load.ID = fmt.Sprintf("%v", id)
	} else {
		load.ID = fmt.Sprintf("temp-%d", time.Now().Unix())
	}
	load.Status = "active"
	load.CreatedAt = time.Now().Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(load)
}

func handleDeleteLoad(w http.ResponseWriter, r *http.Request) {
	displayId := r.URL.Query().Get("id")
	if displayId == "" {
		http.Error(w, "ID parameter required", http.StatusBadRequest)
		return
	}

	token, err := getTurvoToken()
	if err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Get all shipments to find the numeric ID for this display ID
	shipmentsReq, _ := http.NewRequest("GET", "https://my-sandbox.turvo.com/api/shipments/list", nil)
	shipmentsReq.Header.Set("Authorization", "Bearer "+token)
	shipmentsReq.Header.Set("Accept", "application/json")

	client := &http.Client{}
	shipmentsResp, err := client.Do(shipmentsReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get shipments: %v", err), http.StatusInternalServerError)
		return
	}
	defer shipmentsResp.Body.Close()

	var turvoResp TurvoShipmentsResponse
	json.NewDecoder(shipmentsResp.Body).Decode(&turvoResp)

	// Find the numeric ID for this display ID
	var numericId string
	for _, shipment := range turvoResp.Shipments {
		if shipment.ProjectFields.Title.DisplayID == displayId {
			numericId = fmt.Sprintf("%v", shipment.ID)
			break
		}
	}

	if numericId == "" {
		http.Error(w, fmt.Sprintf("Shipment not found for displayId: %s", displayId), http.StatusNotFound)
		return
	}

	// Delete using the numeric ID
	deleteReq, _ := http.NewRequest("DELETE", "https://my-sandbox.turvo.com/api/shipments/"+numericId, nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(deleteReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete shipment: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Load deleted successfully"}`))
		return
	}

	// Return error details for debugging
	http.Error(w, fmt.Sprintf("Delete failed (status %d): %s. Tried numeric ID: %s for displayId: %s", resp.StatusCode, string(body), numericId, displayId), resp.StatusCode)
}
