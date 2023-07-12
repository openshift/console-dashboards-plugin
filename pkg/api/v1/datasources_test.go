package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	datasources "github.com/openshift/console-dashboards-plugin/pkg/datasources"
)

func TestCreateDashboardsHandler(t *testing.T) {
	datasourceManager := datasources.NewDatasourceManager()

	datasourceManager.SetDatasource("test-datasource_name123", &datasources.DataSource{
		Kind: "Prometheus",
		Metadata: datasources.DatasourceMetadata{
			Name:      "test-datasource",
			Namespace: "test-namespace",
		},
	})

	req, err := http.NewRequest("GET", "/api/v1/datasources/test-datasource_name123", nil)
	if err != nil {
		t.Fatal(err)
	}

	reqRecorder := httptest.NewRecorder()
	r := mux.NewRouter()

	r.HandleFunc("/api/v1/datasources/{name}", CreateDashboardsHandler(datasourceManager))

	r.ServeHTTP(reqRecorder, req)

	if status := reqRecorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestCreateDashboardsHandlerInvalidName(t *testing.T) {
	datasourceManager := datasources.NewDatasourceManager()

	datasourceManager.SetDatasource("test", &datasources.DataSource{
		Kind: "Prometheus",
		Metadata: datasources.DatasourceMetadata{
			Name:      "test",
			Namespace: "test-namespace",
		},
	})

	req, err := http.NewRequest("GET", "/api/v1/datasources/invalid%5c", nil)
	if err != nil {
		t.Fatal(err)
	}

	reqRecorder := httptest.NewRecorder()
	r := mux.NewRouter()

	r.HandleFunc("/api/v1/datasources/{name}", CreateDashboardsHandler(datasourceManager))

	r.ServeHTTP(reqRecorder, req)

	if status := reqRecorder.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestCreateDashboardsHandlerInvalidLongName(t *testing.T) {
	datasourceManager := datasources.NewDatasourceManager()

	// Invalid DNS names are longer than 256 characters
	longName := ""
	for i := 0; i < 260; i++ {
		longName += "a"
	}

	datasourceManager.SetDatasource(longName, &datasources.DataSource{
		Kind: "Prometheus",
		Metadata: datasources.DatasourceMetadata{
			Name:      longName,
			Namespace: "test-namespace",
		},
	})

	req, err := http.NewRequest("GET", `/api/v1/datasources/`+longName, nil)
	if err != nil {
		t.Fatal(err)
	}

	reqRecorder := httptest.NewRecorder()
	r := mux.NewRouter()

	r.HandleFunc("/api/v1/datasources/{name}", CreateDashboardsHandler(datasourceManager))

	r.ServeHTTP(reqRecorder, req)

	if status := reqRecorder.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}
