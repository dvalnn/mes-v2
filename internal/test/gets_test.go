package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	
	sim "mes/internal/sim"
	net_erp "mes/internal/net/erp"
)

func TestGetShipmentArrivals(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET method, got %s", r.Method)
		}

		if r.URL.Path != "/materials/expected" {
			t.Fatalf("expected /materials/arrivals path, got %s", r.URL.Path)
		}

		if val := r.URL.Query().Get("day"); val != "1" {
			t.Fatalf("expected day 1, got %s", val)
		}

		w.Write([]byte(`[
      { "material_type": "P1", "shipment_id": 1, "quantity": 10 },
      { "material_type": "P2", "shipment_id": 2, "quantity": 20 }
      ]`))
	}

	handler := http.HandlerFunc(handlerFunc)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx := getHttpTestContext(server.URL, net_erp.DEFAULT_HTTP_TIMEOUT)
	shipments, err := sim.GetShipments(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(shipments) != 2 {
		t.Fatalf("expected 2 shipments, got %d", len(shipments))
	}

	expected := []sim.Shipment{
		{MaterialKind: "P1", ID: 1, NPieces: 10},
		{MaterialKind: "P2", ID: 2, NPieces: 20},
	}

	for i, s := range shipments {
		if s != expected[i] {
			t.Errorf("expected %+v, got %+v", expected[i], s)
		}
	}
}

func TestGetProduction(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/production" {
			t.Errorf("expected /production path, got %s", r.URL.Path)
		}
		if val := r.URL.Query().Get("max_n_items"); val != "2" {
			t.Errorf("expected max_n_items 2, got %s", val)
		}

		w.Write([]byte(
			`[ { "steps": [ {
                  "material_id": "mat01",
                  "product_id": "mat02",
                  "material_kind": "P3",
                  "product_kind": "P4",
                  "tool": "T1",
                  "transformation_id": 123,
                  "operation_time": 300
                }] },
            { "steps": [ {
                  "material_id": "string",
                  "product_id": "string",
                  "material_kind": "string",
                  "product_kind": "string",
                  "tool": "string",
                  "transformation_id": 456,
                  "operation_time": 240
                } ] }
          ]`))
	}
	handler := http.HandlerFunc(handlerFunc)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx := getHttpTestContext(server.URL, net_erp.DEFAULT_HTTP_TIMEOUT)
	recipes, err := sim.GetPieces(ctx, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(recipes) != 2 {
		t.Fatalf("expected 2 recipes, got %d", len(recipes))
	}

	expected := []sim.Piece{
		{
			Steps: []sim.Transformation{
				{
					MaterialID:   "mat01",
					ProductID:    "mat02",
					MaterialKind: "P3",
					ProductKind:  "P4",
					Tool:         "T1",
					ID:           123,
					Time:         300,
				},
			},
		},
		{
			Steps: []sim.Transformation{
				{
					MaterialID:   "string",
					ProductID:    "string",
					MaterialKind: "string",
					ProductKind:  "string",
					Tool:         "string",
					ID:           456,
					Time:         240,
				},
			},
		},
	}

	for i, r := range recipes {
		for j, tf := range r.Steps {
			if tf != expected[i].Steps[j] {
				t.Errorf("expected %+v, got %+v", expected[i].Steps[j], tf)
			}
		}
	}
}
