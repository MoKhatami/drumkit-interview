package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Load struct {
	ExternalTMSLoadID string         `json:"externalTMSLoadID"`
	FreightLoadID     string         `json:"freightLoadID,omitempty"`
	Status            string         `json:"status"`
	Customer          Customer       `json:"customer"`
	Pickup            Location       `json:"pickup,omitempty"`
	Consignee         Location       `json:"consignee,omitempty"`
	Broker            Broker         `json:"broker,omitempty"`
	Specifications    Specifications `json:"specifications,omitempty"`
}

type Customer struct {
	Name string `json:"name"`
}

type Location struct {
	Name    string `json:"name,omitempty"`
	City    string `json:"city"`
	State   string `json:"state"`
	Country string `json:"country,omitempty"`
}

type Broker struct {
	Name string `json:"name,omitempty"`
}

type Specifications struct {
	TotalWeight    float64 `json:"totalWeight,omitempty"`
	NumCommodities int     `json:"numCommodities,omitempty"`
	RouteMiles     float64 `json:"routeMiles,omitempty"`
}

type CreateLoadRequest struct {
	Customer        string `json:"customer"`
	Pickup          string `json:"pickup"`
	PickupState     string `json:"pickupState"`
	PickupCountry   string `json:"pickupCountry"`
	Delivery        string `json:"delivery"`
	DeliveryState   string `json:"deliveryState"`
	DeliveryCountry string `json:"deliveryCountry"`
}

// Global token variable
var turvoToken string

func getTurvoToken() string {
	// OAuth request payload
	payload := map[string]string{
		"grant_type":    "password",
		"client_id":     os.Getenv("TURVO_CLIENT_ID"),
		"client_secret": os.Getenv("TURVO_CLIENT_SECRET"),
		"username":      os.Getenv("TURVO_USERNAME"),
		"password":      os.Getenv("TURVO_PASSWORD"),
		"scope":         "read+trust+write",
		"type":          "business",
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
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	return result["access_token"].(string)
}

func deleteLoad(c *gin.Context) {
	id := c.Param("id")

	req, _ := http.NewRequest("DELETE", os.Getenv("TURVO_BASE_URL")+"/api/shipments/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+turvoToken)

	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	c.JSON(200, gin.H{"success": true})
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: Error loading .env file")
	}

	// Get token on startup
	turvoToken = getTurvoToken()

	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/api/loads", listLoads)
	r.POST("/api/loads", createLoad)
	r.DELETE("/api/loads/:id", deleteLoad)
	r.Run(":8080")
}

func listLoads(c *gin.Context) {
	apiURL := fmt.Sprintf("%s/api/shipments/list", os.Getenv("TURVO_BASE_URL"))
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+turvoToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	var turvoResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&turvoResponse)

	var allShipments []interface{}
	if directShipments, ok := turvoResponse["shipments"].([]interface{}); ok {
		allShipments = directShipments
	}

	loads := make([]Load, 0)

	for _, shipment := range allShipments {
		s := shipment.(map[string]interface{})

		id := fmt.Sprintf("%.0f", s["id"].(float64))
		// Get customer and broker from correct locations
		customer := Customer{Name: "Unknown"}
		broker := Broker{Name: "Unknown"}

		// Extract customer from customer_orders
		if details, ok := s["details"].(map[string]interface{}); ok {
			if customerOrders, ok := details["customer_orders"].([]interface{}); ok && len(customerOrders) > 0 {
				if firstOrder, ok := customerOrders[0].(map[string]interface{}); ok {
					if customerObj, ok := firstOrder["customer"].(map[string]interface{}); ok {
						if name, ok := customerObj["name"].(string); ok {
							customer.Name = name
						}
					}
				}
			}

			// Extract broker from contributors
			if contributors, ok := details["contributors"].([]interface{}); ok {
				for _, contrib := range contributors {
					if c, ok := contrib.(map[string]interface{}); ok {
						if contributorUser, ok := c["contributorUser"].(map[string]interface{}); ok {
							if name, ok := contributorUser["name"].(string); ok {
								if title, ok := c["title"].(map[string]interface{}); ok {
									if role, ok := title["value"].(string); ok && role == "Broker" {
										broker.Name = name
									}
								}
							}
						}
					}
				}
			}
		}

		// Fallback: try projectFields.title.customer for customer info
		if customer.Name == "Unknown" {
			if projectFields, ok := s["projectFields"].(map[string]interface{}); ok {
				if title, ok := projectFields["title"].(map[string]interface{}); ok {
					if customers, ok := title["customer"].([]interface{}); ok && len(customers) > 0 {
						if firstCustomer, ok := customers[0].(map[string]interface{}); ok {
							if name, ok := firstCustomer["name"].(string); ok {
								customer.Name = name
							}
						}
					}
				}
			}
		}

		pickup := "Unknown"
		delivery := "Unknown"
		if projectFields, ok := s["projectFields"].(map[string]interface{}); ok {
			if route, ok := projectFields["route"].(map[string]interface{}); ok {
				if start, ok := route["start"].(string); ok {
					pickup = start
				}
				if end, ok := route["end"].(string); ok {
					delivery = end
				}
			}
		}

		status := "Unknown"
		if projectFields, ok := s["projectFields"].(map[string]interface{}); ok {
			if statusObj, ok := projectFields["status"].(map[string]interface{}); ok {
				if desc, ok := statusObj["description"].(string); ok {
					status = desc
				}
			}
		}

		load := Load{
			ExternalTMSLoadID: id,
			Status:            status,
			Customer:          customer,
			Pickup:            Location{City: pickup},
			Consignee:         Location{City: delivery},
			Broker:            broker,
		}

		loads = append(loads, load)
	}

	c.JSON(200, gin.H{
		"loads":      loads,
		"pagination": turvoResponse["pagination"],
	})
}

func createLoad(c *gin.Context) {
	var req CreateLoadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Create Turvo shipment payload with dynamic locations
	payload := map[string]interface{}{
		"ltlShipment": false,
		"startDate": map[string]interface{}{
			"date":     "2025-09-05T08:00:00Z",
			"timeZone": "America/New_York",
		},
		"endDate": map[string]interface{}{
			"date":     "2025-09-06T17:00:00Z",
			"timeZone": "America/New_York",
		},
		"lane": map[string]interface{}{
			"start": req.Pickup + ", " + req.PickupState + ", " + req.PickupCountry,
			"end":   req.Delivery + ", " + req.DeliveryState + ", " + req.DeliveryCountry,
		},
		"customerOrder": []map[string]interface{}{
			{
				"customer": map[string]interface{}{
					"name": req.Customer,
					"id":   req.Customer,
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

	// Call Turvo create API
	createURL := fmt.Sprintf("%s/api/shipments", os.Getenv("TURVO_BASE_URL"))
	httpReq, _ := http.NewRequest("POST", createURL, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Authorization", "Bearer "+turvoToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(httpReq)
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("DEBUG - Shipment creation status: %d\n", resp.StatusCode)
	fmt.Printf("DEBUG - Shipment creation response: %+v\n", result)

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		c.JSON(200, gin.H{"success": true, "message": "Load created successfully"})
	} else {
		fmt.Printf("DEBUG - Shipment creation failed. Status: %d\n", resp.StatusCode)
		c.JSON(resp.StatusCode, result)
	}
}
