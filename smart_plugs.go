package main

import (
	"context"
	"encoding/json"
	"github.com/tess1o/tapo-go"
	"log/slog"
)

type SmartPlug struct {
	Name     string              `json:"name"`
	Host     string              `json:"host"`
	Client   *tapo.SmartPlug     `json:"-"`
	Exporter *PrometheusExporter `json:"-"`
	Username string              `json:"-"`
	Password string              `json:"-"`
}

func (s SmartPlug) GetEnergyUsage() (*tapo.EnergyUsageResponse, error) {
	return s.Client.GetEnergyUsage(context.Background())
}

func (s SmartPlug) GetEmeterData() (*tapo.EmeterData, error) {
	return s.Client.GetEmeterData(context.Background())
}

func initSmartPlugs(devices []*SmartPlug, username string, password string, exporter *PrometheusExporter) {
	for i := range devices {
		devices[i].Exporter = exporter
		devices[i].Username = username
		devices[i].Password = password
		client, err := initSmartPlugClient(devices[i])
		if err != nil {
			slog.Error("Failed to create Tapo client", "error", err, "device", devices[i].Name)
			continue
		}
		devices[i].Client = client
	}
}

func initSmartPlugClient(plug *SmartPlug) (*tapo.SmartPlug, error) {
	slog.Info("Creating Tapo client", "host", plug.Host)
	return tapo.NewSmartPlug(context.Background(), plug.Host, plug.Username, plug.Password, tapo.Options{
		RetryConfig: tapo.DefaultRetryConfig,
	})
}

func handleSmartPlug(device *SmartPlug) {
	if device.Client == nil {
		client, err := initSmartPlugClient(device)
		if err != nil {
			slog.Error("Failed to create Tapo client", "error", err, "device", device.Name)
			return
		}
		device.Client = client
	}
	r, err := device.GetEnergyUsage()
	if err != nil {
		slog.Error("Error getting energy usage", "device", device.Name, "error", err)
		client, initErr := initSmartPlugClient(device)
		if initErr != nil {
			slog.Error("Failed to create Tapo client", "error", initErr, "device", device.Name)
			return
		}
		device.Client = client
	} else {
		slog.Info("Successfully received metrics", "device", device.Name)
		d, _ := json.Marshal(r.Result)
		var params map[string]interface{}
		json.Unmarshal(d, &params)
		device.Exporter.HandleSmartPlug(context.Background(), device, params)
		handleSmartPlugEmeter(device)
	}
}

func handleSmartPlugEmeter(device *SmartPlug) {
	r, err := device.GetEmeterData()
	if err != nil {
		slog.Warn("Error getting emeter data", "device", device.Name, "error", err)
		return
	}
	d, _ := json.Marshal(r.Result)
	var params map[string]interface{}
	json.Unmarshal(d, &params)
	device.Exporter.HandleSmartPlug(context.Background(), device, params)
}
