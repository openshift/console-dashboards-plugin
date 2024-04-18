package datasources

import (
	"context"
	"net/http/httputil"
	"sync"
	"time"

	logrus "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var log = logrus.WithField("module", "datasources")

type DataSourceMap = map[string]*DataSource
type CAMap = map[string]*string
type ProxiesMap = map[string]*httputil.ReverseProxy

type DatasourceManager struct {
	datasourceMap *DataSourceMap
	caMap         *CAMap
	proxiesMap    *ProxiesMap
	mutex         *sync.Mutex
}

func NewDatasourceManager() *DatasourceManager {
	return &DatasourceManager{
		datasourceMap: &DataSourceMap{},
		caMap:         &CAMap{},
		proxiesMap:    &ProxiesMap{},
		mutex:         &sync.Mutex{}}
}

func (manager *DatasourceManager) SetDatasource(datasourceName string, datasource *DataSource) {
	manager.mutex.Lock()
	(*manager.datasourceMap)[datasourceName] = datasource
	// Set the proxy to nil so that it will be recreated
	(*manager.proxiesMap)[datasourceName] = nil
	manager.mutex.Unlock()
}

func (manager *DatasourceManager) GetDatasource(datasourceName string) *DataSource {
	manager.mutex.Lock()
	defer func() {
		manager.mutex.Unlock()
	}()
	return (*manager.datasourceMap)[datasourceName]
}

func (manager *DatasourceManager) SetCA(datasourceName string, ca *string) {
	manager.mutex.Lock()
	(*manager.caMap)[datasourceName] = ca
	// Set the proxy to nil so that it will be recreated
	(*manager.proxiesMap)[datasourceName] = nil
	manager.mutex.Unlock()
}

func (manager *DatasourceManager) GetCA(datasourceName string) *string {
	manager.mutex.Lock()
	defer func() {
		manager.mutex.Unlock()
	}()
	return (*manager.caMap)[datasourceName]
}

func (manager *DatasourceManager) GetProxy(datasourceName string) *httputil.ReverseProxy {
	manager.mutex.Lock()
	defer func() {
		manager.mutex.Unlock()
	}()
	return (*manager.proxiesMap)[datasourceName]
}

func (manager *DatasourceManager) SetProxy(datasourceName string, proxy *httputil.ReverseProxy) {
	manager.mutex.Lock()
	(*manager.proxiesMap)[datasourceName] = proxy
	manager.mutex.Unlock()
}

func (manager *DatasourceManager) Delete(datasourceName string) {
	manager.mutex.Lock()
	delete(*manager.proxiesMap, datasourceName)
	delete(*manager.datasourceMap, datasourceName)
	delete(*manager.caMap, datasourceName)
	manager.mutex.Unlock()
}

func (manager *DatasourceManager) WatchDatasources(namespace string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.WithError(err).Error("cannot get in cluster config")
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.WithError(err).Error("cannot create k8s client")
		return err
	}

	labelSelector := labels.SelectorFromSet(labels.Set{"console.openshift.io/dashboard-datasource": "true"})

	log.Info("watching datasources")

	for {
		watcher, err := client.CoreV1().ConfigMaps(namespace).Watch(context.Background(), metav1.ListOptions{LabelSelector: labelSelector.String()})

		if err != nil {
			log.WithError(err).Error("unable to create datasources watcher, will retry in 5 minutes")
			time.Sleep(time.Minute * 5)
			continue
		}

		for {
			event, open := <-watcher.ResultChan()
			if open {
				switch event.Type {
				case watch.Added:
					fallthrough
				case watch.Modified:
					if configMap, ok := event.Object.(*v1.ConfigMap); ok {
						dataSourceYaml, ok := configMap.Data["dashboard-datasource.yaml"]
						if !ok {
							log.Errorf("key 'dashboard-datasource.yaml' not found in configMap: %s", configMap.Name)
							continue
						}

						var configMapData DataSource
						err := yaml.Unmarshal([]byte(dataSourceYaml), &configMapData)

						if err != nil {
							log.WithError(err).Errorf("cannot unmarshall configmap datasource in key 'dashboard-datasource.yaml': %s", configMap.Name)
							continue
						}
						manager.SetDatasource(configMapData.Metadata.Name, &configMapData)
						log.WithField("datasource_name", configMapData.Metadata.Name).Infof("datasource loaded: %s", configMapData.Metadata.Name)

						caValue, ok := configMap.Data["dashboard-datasource-ca"]

						if ok {
							manager.SetCA(configMapData.Metadata.Name, &caValue)
							log.WithField("datasource_name", configMapData.Metadata.Name).Infof("CA loaded: %s", configMapData.Metadata.Name)
						}
					} else {
						log.Debugf("failed when modified %v", event.Object)
					}
				case watch.Deleted:
					if configMap, ok := event.Object.(*v1.ConfigMap); ok {
						dataSourceYaml, ok := configMap.Data["dashboard-datasource.yaml"]
						if !ok {
							log.Errorf("key 'dashboard-datasource.yaml' not found in configMap: %s", configMap.Name)
							continue
						}

						var configMapData DataSource
						err := yaml.Unmarshal([]byte(dataSourceYaml), &configMapData)
						if err != nil {
							log.WithError(err).Errorf("cannot unmarshall configmap: %s while beign deleted", configMap.Name)
							continue
						}

						manager.Delete(configMapData.Metadata.Name)
						log.WithField("datasource-name", configMapData.Metadata.Name).Infof("datasource deleted: %s", configMapData.Metadata.Name)
					} else {
						log.Debugf("failed when deleted %v", event.Object)
					}
				default:
					// Do nothing
				}
			} else {
				// watch channel exhausted, break to watch again
				break
			}
		}

		time.Sleep(time.Second * 10)
	}
}
