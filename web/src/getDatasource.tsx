import * as LRU from 'lru-cache';
const BACKEND_API = '/api/proxy/plugin/console-dashboards-plugin/backend';

type DatasourceInfo = {
  basePath: string;
  dataSourceType: string;
};

const cache = new LRU<string, DatasourceInfo>({
  max: 500,
  ttl: 1000 * 60 * 5,
});

// // NOTE: image jezhu/console-dashboards-plugin:0.0.1
// async function getDataSource(datasourceName: string): Promise<DatasourceInfo> {
//   if (cache.has(datasourceName)) {
//     return cache.get(datasourceName);
//   }

//   // NOTE : this doesn't work for local testing -- we're not serveing the endpoint from localhost:9000, and if I replace with actual host there is a CORS auth error 
//   const datasource = await fetch(
//     `${BACKEND_API}/api/v1/datasources/${datasourceName}`,
//   );

//   // const datasource = await fetch(
//   //   `https://console-openshift-console.apps.rhoms-4.14-080704.dev.openshiftappsvc.org/api/proxy/plugin/console-dashboards-plugin/backend/api/v1/datasources/cluster-prometheus-proxy`
//   // )

//   try {
//     const jsonData = await datasource.json();
//     const dataSourceType = jsonData?.spec?.plugin?.kind;

//     const basePath = `/api/proxy/plugin/console-dashboards-plugin/backend/proxy/${datasourceName}`;
    
//     const datasourceInfo = { basePath, dataSourceType };
//     cache.set(datasourceName, datasourceInfo);

//     return datasourceInfo;
//   } catch (err) {
//     console.error(err);
//   }
  
//   return null;
// }



// NOTE: for local dev -- replaces the fetch(customDataSourceType) method with hard-code dataSourceType="prometheus"
async function getDataSource(datasourceName: string): Promise<DatasourceInfo> {
  if (cache.has(datasourceName)) {
    return cache.get(datasourceName);
  }

  try {
    const dataSourceType = "prometheus"

    const basePath = `/api/proxy/plugin/console-dashboards-plugin/backend/proxy/${datasourceName}`;
    
    const datasourceInfo = { basePath, dataSourceType };
    cache.set(datasourceName, datasourceInfo);

    return datasourceInfo;
  } catch (err) {
    console.error(err);
  }
  
  return null;
}




// // TODO: clean up this is for testing 
// // NOTE: image jezhu/console-dashboards-plugin:0.0.2
// function getDataSource(datasourceName: string): DatasourceInfo {
//   if (cache.has(datasourceName)) {
//     return cache.get(datasourceName);
//   }

//   // const datasource =  fetch(
//   //   `${BACKEND_API}/api/v1/datasources/${datasourceName}`,
//   // );

//   try {
//     // const jsonData =  datasource.json();
//     // const dataSourceType = jsonData?.spec?.plugin?.kind;

//     // expected JSON object is 
//     // {"kind":"Datasource","metadata":{"name":"cluster-prometheus-proxy"},"spec":{"plugin":{"kind":"prometheus","spec":{"direct_url":""}}}}
//     // need to spoof jsonData becuase it fetches localhost:9000 instead of https://console-openshift-console.apps.rhoms-4.13-073104.dev.openshiftappsvc.org/

//     // console.log("JZ hello world")
//     const basePath = `/api/proxy/plugin/console-dashboards-plugin/backend/proxy/${datasourceName}`;
//     const dataSourceType = "prometheus";

//     const datasourceInfo = { basePath, dataSourceType };

//     cache.set(datasourceName, datasourceInfo);

//     console.warn("JZ console-dashboards-plugin dataSourceInfo: ", datasourceInfo)

//     return datasourceInfo;
//   } catch (err) {
//     console.error(err);
//   }

//   console.warn("JZ console-dashboards-plugin returns null")

//   return null;
// }



// NOTES: 0.0.3
// function getDataSource(datasourceName: string): Promise<DatasourceInfo> {
//   if (cache.has(datasourceName)) {
//     return cache.get(datasourceName);
//   }

//   console.warn("beans in getDataSource")

//   fetch(`${BACKEND_API}/api/v1/datasources/${datasourceName}`)
//     .then(async (res) => {
//       if (res.ok) {
//         console.warn("beans res: ", res)
//         const jsonData = await res.json();
//         console.warn("beans jsonData: ", jsonData)
//         const dataSourceType = jsonData?.spec?.plugin?.kind;
//         console.warn("beans dataSourceType: ", dataSourceType)
//         const basePath = `/api/proxy/plugin/console-dashboards-plugin/backend/proxy/${datasourceName}`;
//         const datasourceInfo = { basePath, dataSourceType };
//         console.warn("beans datasourceInfo: ", JSON.stringify(datasourceInfo))
//         cache.set(datasourceName, datasourceInfo);
//         return datasourceInfo;
//       } else {
//         throw new Error('Invalid response');
//       }
//     })
//     .catch((err) => {
//       console.error(err);
//     })

//   return null;
// }

// function getDataSource(datasourceName: string): Promise<DatasourceInfo> {
//   if (cache.has(datasourceName)) {
//     return cache.get(datasourceName);
//   }

//   fetch(`${BACKEND_API}/api/v1/datasources/${datasourceName}`)
//     .then(async (res) => {
//       if (res.ok) {
//         console.warn("beans res: ", res)
//         const jsonData = await res.json();
//         console.warn("beans jsonData: ", jsonData)
//         const dataSourceType = jsonData?.spec?.plugin?.kind;
//         console.warn("beans dataSourceType: ", dataSourceType)
//         const basePath = `/api/proxy/plugin/console-dashboards-plugin/backend/proxy/${datasourceName}`;
//         const datasourceInfo = { basePath, dataSourceType };
//         console.warn("beans datasourceInfo: ", JSON.stringify(datasourceInfo))
//         cache.set(datasourceName, datasourceInfo);
//         return datasourceInfo;
//       } else {
//         throw new Error('Invalid response');
//       }
//     })
//     .catch((err) => {
//       console.error(err);
//     })
  
  

//   return null;
// }



export default getDataSource;


