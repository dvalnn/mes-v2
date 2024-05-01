package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mes "mes/internal"
)

func TestPostTimeout(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
	}
	handler := http.HandlerFunc(handlerFunc)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx := getHttpTestContext(server.URL, time.Millisecond)
	form := mes.DateForm{Day: 2}

	if err := form.Post(ctx); err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestPostError(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	handler := http.HandlerFunc(handlerFunc)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx := getHttpTestContext(server.URL, mes.DEFAULT_HTTP_TIMEOUT)
	form := mes.DateForm{Day: 2}

	if err := form.Post(ctx); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestPostCurrentDate(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/date" {
			t.Errorf("expected /date path, got %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("error parsing form: %v", err)
		}
		if r.FormValue("day") != "2" {
			t.Errorf("expected day 2, got %s", r.FormValue("day"))
		}

		w.WriteHeader(http.StatusCreated)
	}

	handler := http.HandlerFunc(handlerFunc)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx := getHttpTestContext(server.URL, mes.DEFAULT_HTTP_TIMEOUT)

	form := mes.DateForm{Day: 2}
	if err := form.Post(ctx); err != nil {
		t.Errorf("error posting date: %v", err)
	}
}

func TestPostWarehouseExit(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/warehouse" {
			t.Errorf("expected /warehouse path, got %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("error parsing form: %v", err)
		}
		if r.FormValue("item_id") != "1" {
			t.Errorf("expected item_id 1, got %s", r.FormValue("item_id"))
		}
		if r.FormValue("exit") != "L1" {
			t.Errorf("expected exit to L1, got %s", r.FormValue("exit"))
		}

		w.WriteHeader(http.StatusCreated)
	}

	handler := http.HandlerFunc(handlerFunc)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx := getHttpTestContext(server.URL, mes.DEFAULT_HTTP_TIMEOUT)

	warehouseTest := mes.WarehouseExitForm{ItemId: "1", LineId: mes.ID_L1}
	if err := warehouseTest.Post(ctx); err != nil {
		t.Errorf("error posting warehouse exit: %v", err)
	}
}

func TestPostWarehouseEntry(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/warehouse" {
			t.Errorf("expected /warehouse path, got %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("error parsing form: %v", err)
		}
		if r.FormValue("item_id") != "1" {
			t.Errorf("expected item_id 1, got %s", r.FormValue("item_id"))
		}
		if r.FormValue("entry") != "W1" && r.FormValue("entry") != "W2" {
			t.Errorf("expected entry to W1 or W2 got %s", r.FormValue("entry"))
		}
		// send back a status created
		w.WriteHeader(http.StatusCreated)
	}
	handler := http.HandlerFunc(handlerFunc)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx := getHttpTestContext(server.URL, mes.DEFAULT_HTTP_TIMEOUT)
	warehouseTest := mes.WarehouseEntryForm{ItemId: "1", WarehouseId: mes.ID_W1}
	if err := warehouseTest.Post(ctx); err != nil {
		t.Errorf("error posting warehouse entry: %v", err)
	}
}

func TestPostTransformationCompletion(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/transformations" {
			t.Errorf("expected /transformations path, got %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("error parsing form: %v", err)
		}
		if r.FormValue("transf_id") != "1" {
			t.Errorf("expected transformation_id 1, got %s", r.FormValue("transformation_id"))
		}
		if r.FormValue("time_taken") != "2" {
			t.Errorf("expected time_taken 2, got %s", r.FormValue("time_taken"))
		}
		if r.FormValue("material_id") != "1" {
			t.Errorf("expected material_id 1, got %s", r.FormValue("material_id"))
		}
		if r.FormValue("product_id") != "2" {
			t.Errorf("expected product_id 2, got %s", r.FormValue("product_id"))
		}
		if r.FormValue("line_id") != "L1" {
			t.Errorf("expected line_id L1, got %s", r.FormValue("line_id"))
		}

		w.WriteHeader(http.StatusCreated)
	}

	handler := http.HandlerFunc(handlerFunc)
	server := httptest.NewServer(handler)
	defer server.Close()

	form := mes.TransfCompletionForm{
		MaterialID:       "1",
		ProductID:        "2",
		LineID:           mes.ID_L1,
		TransformationID: 1,
		TimeTaken:        2,
	}

	ctx := getHttpTestContext(server.URL, mes.DEFAULT_HTTP_TIMEOUT)
	if err := form.Post(ctx); err != nil {
		t.Errorf("error posting transformation completion: %v", err)
	}
}
