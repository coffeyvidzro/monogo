import type { Graph, Thing, WithContext } from "schema-dts";

const BASE_URL = "https://leamout.com";

export function constructMetagraph(): Graph {
  return {
    "@context": "https://schema.org",
    "@graph": [
      {
        "@type": "Organization",
        "@id": `${BASE_URL}/#organization`,
        name: "Leamout",
        url: BASE_URL,
        logo: `${BASE_URL}/logo.png`,
      },
      {
        "@type": "WebSite",
        "@id": `${BASE_URL}/#website`,
        name: "Leamout",
        url: BASE_URL,
        publisher: {
          "@id": `${BASE_URL}/#organization`,
        },
      },
      {
        "@type": "WebApplication",
        "@id": `${BASE_URL}/#application`,
        name: "Leamout",
        url: BASE_URL,
        applicationCategory: "BusinessApplication",
        operatingSystem: "Web",
        publisher: {
          "@id": `${BASE_URL}/#organization`,
        },
      },
    ],
  };
}

export function serializeSchema(schema: Graph | WithContext<Thing>): string {
  return JSON.stringify(schema)
    .replace(/</g, "\\u003c")
    .replace(/>/g, "\\u003e")
    .replace(/&/g, "\\u0026")
    .replace(/'/g, "\\u0027");
}
