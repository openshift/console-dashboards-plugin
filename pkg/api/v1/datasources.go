package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jgbernalp/dashboards-datasource-plugin/pkg/datasources"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("module", "datasources-api")

func CreateDashboardsHandler(datasourceManager *datasources.DatasourceManager) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		datasourceName := vars["name"]

		if len(datasourceName) == 0 {
			log.Error("invalid datasource name")
			http.Error(w, "invalid datasource name", http.StatusBadRequest)
			return
		}

		datasource := datasourceManager.GetDatasource(datasourceName)

		if datasource == nil {
			log.Errorf("datasource not found: %s", datasourceName)
			http.Error(w, "datasource not found", http.StatusNotFound)
			return
		}

		datasourceData, err := json.Marshal(datasource)
		if err != nil {
			log.WithError(err).Error("cannot marshal datasource info")
			http.Error(w, "cannot marshal datasource info", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(datasourceData)
	}
}
