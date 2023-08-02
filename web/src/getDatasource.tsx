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

// async function getDataSource(datasourceName: string): Promise<DatasourceInfo> {
//   if (cache.has(datasourceName)) {
//     return cache.get(datasourceName);
//   }

//   const datasource = await fetch(
//     `${BACKEND_API}/api/v1/datasources/${datasourceName}`,
//   );

//   try {
//     const jsonData = await datasource.json();

//     const basePath = `/api/proxy/plugin/console-dashboards-plugin/backend/proxy/${datasourceName}`;
//     const dataSourceType = jsonData?.spec?.plugin?.kind;

//     const datasourceInfo = { basePath, dataSourceType };

//     cache.set(datasourceName, datasourceInfo);

//     return datasourceInfo;
//   } catch (err) {
//     console.error(err);
//   }

//   return null;
// }

// export default getDataSource;



// TODO: clean up this is for testing 
function getDataSource(datasourceName: string): DatasourceInfo {
  if (cache.has(datasourceName)) {
    return cache.get(datasourceName);
  }

  // const datasource =  fetch(
  //   `${BACKEND_API}/api/v1/datasources/${datasourceName}`,
  // );

  try {
    // const jsonData =  datasource.json();
    // const dataSourceType = jsonData?.spec?.plugin?.kind;

    // expected JSON object is 
    // {"kind":"Datasource","metadata":{"name":"cluster-prometheus-proxy"},"spec":{"plugin":{"kind":"prometheus","spec":{"direct_url":""}}}}
    // need to spoof jsonData becuase it fetches localhost:9000 instead of https://console-openshift-console.apps.rhoms-4.13-073104.dev.openshiftappsvc.org/

    console.log("JZ hello world")
    const basePath = `/api/proxy/plugin/console-dashboards-plugin/backend/proxy/${datasourceName}`;
    const dataSourceType = "prometheus";

    const datasourceInfo = { basePath, dataSourceType };

    cache.set(datasourceName, datasourceInfo);

    return datasourceInfo;
  } catch (err) {
    console.error(err);
  }

  return null;
}

export default getDataSource;
