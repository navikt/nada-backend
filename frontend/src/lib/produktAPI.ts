import * as z from "zod";
export const BACKEND_ENDPOINT =
  process.env.NODE_ENV === "development" ? "http://localhost:8080" : "";
export const API_ROOT = `${BACKEND_ENDPOINT}/api/v1`;

const DataProduktTilgangOppdateringSchema = z.object({
  subject: z.string(),
  expires: z.string().nullable(),
  type: z.string(),
});

const DataProduktTilgangResponseSchema = z.object({
  dataproduct_id: z.string(),
  author: z.string(),
  subject: z.string(),
  action: z.string(),
  time: z.string(),
  expires: z.string(),
});

export const DataLagerBucketSchema = z.object({
  type: z.literal("bucket"),
  project_id: z.string(),
  bucket_id: z.string(),
});

export const DataLagerBigquerySchema = z.object({
  type: z.literal("bigquery"),
  project_id: z.string(),
  dataset_id: z.string(),
  resource_id: z.string(),
});

export const DataLagerSchema = z.union([
  DataLagerBucketSchema,
  DataLagerBigquerySchema,
]);
export type DataLagerBucket = z.infer<typeof DataLagerBucketSchema>;
export type DataLagerBigquery = z.infer<typeof DataLagerBigquerySchema>;
export type DataLager = z.infer<typeof DataLagerSchema>;

export const DataProduktSchema = z.object({
  name: z.string().nonempty(),
  description: z.string(),
  team: z.string(),
  datastore: DataLagerSchema.array().optional(),
});

const DataProduktPartialSchema = DataProduktSchema.partial();

export const DataProduktResponseSchema = z.object({
  id: z.string(),
  updated: z.string(),
  created: z.string(),
  data_product: DataProduktSchema,
});

export const BrukerInfoSchema = z.object({
  email: z.string(),
  teams: z.array(z.string()),
});

const DataProduktListSchema = DataProduktResponseSchema.array();
const DataProduktTilgangListSchema =
  DataProduktTilgangResponseSchema.array().nullable();

export type DataProdukt = z.infer<typeof DataProduktSchema>;
export type DataProduktPartial = z.infer<typeof DataProduktPartialSchema>;
export type DataProduktResponse = z.infer<typeof DataProduktResponseSchema>;
export type DataProduktListe = z.infer<typeof DataProduktListSchema>;
export type BrukerInfo = z.infer<typeof BrukerInfoSchema>;
export type DataProduktTilgangOppdatering = z.infer<
  typeof DataProduktTilgangOppdateringSchema
>;
export type DataProduktTilgangResponse = z.infer<
  typeof DataProduktTilgangResponseSchema
>;
export type DataProduktTilgangListe = z.infer<
  typeof DataProduktTilgangListSchema
>;

export const hentProdukter = async (): Promise<DataProduktListe> => {
  const res = await fetch(`${API_ROOT}/dataproducts`);
  const json = await res.json();

  return DataProduktListSchema.parse(json);
};
export const hentTilganger = async (
  produktID: string
): Promise<DataProduktTilgangListe> => {
  let res: Response;

  try {
    res = await fetch(`${API_ROOT}/access/${produktID}`, {
      credentials: "include",
    });
  } catch (e) {
    console.log(e);
    throw new Error(`${e}`);
  }

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${await res.text()}`);
  }
  return DataProduktTilgangListSchema.parse(await res.json());
};

export const hentProdukt = async (
  produktID: string
): Promise<DataProduktResponse> => {
  let res: Response;

  try {
    res = await fetch(`${API_ROOT}/dataproducts/${produktID}`, {
      credentials: "include",
    });
  } catch (e) {
    console.log(e);
    throw new Error(`${e}`);
  }

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${await res.text()}`);
  }

  return DataProduktResponseSchema.parse(await res.json());
};

export const slettProdukt = async (produktID: string): Promise<void> => {
  let res: Response;

  try {
    res = await fetch(`${API_ROOT}/dataproducts/${produktID}`, {
      method: "delete",
      credentials: "include",
    });
  } catch (e) {
    console.log(e);
    throw new Error(`Nettverksfeil: ${e}`);
  }

  if (!res.ok) throw res;
};

export const opprettProdukt = async (
  nyttProdukt: DataProdukt
): Promise<string> => {
  const res = await fetch(`${API_ROOT}/dataproducts`, {
    method: "POST",
    credentials: "include",
    body: JSON.stringify(nyttProdukt),
  });

  if (res.status !== 201) {
    throw new Error(`HTTP ${res.status}: ${await res.text()}`);
  }

  return await res.text();
};

export const oppdaterProdukt = async (
  produktID: string,
  produkt: DataProdukt
): Promise<string> => {
  const res = await fetch(`${API_ROOT}/dataproducts/${produktID}`, {
    method: "PUT",
    credentials: "include",
    body: JSON.stringify(produkt),
  });

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${await res.text()}`);
  }

  return await res.text();
};

export const oppdaterTilgang = async (
  produktID: string,
  oppdatertProdukt: DataProduktTilgangOppdatering
): Promise<string> => {
  const res = await fetch(`${API_ROOT}/access/${produktID}`, {
    method: "POST",
    credentials: "include",
    body: JSON.stringify(oppdatertProdukt),
  });

  if (!res.ok) {
    throw new Error(
      `Kunne ikke oppdatere produkt: ${res.status}: ${await res.text()}`
    );
  }

  return await res.text();
};

export const giTilgang = async (
  produkt: DataProduktResponse,
  subject: string,
  expiry: string | null
) => {
  const produktOppdateringer: DataProduktTilgangOppdatering = {
    subject: subject,
    expires: expiry,
    type: "user",
  };

  await oppdaterTilgang(produkt.id, produktOppdateringer);
};

export const hentBrukerInfo = async (): Promise<BrukerInfo> => {
  const res = await fetch(`${API_ROOT}/userinfo`, { credentials: "include" });
  if (!res.ok) throw new Error("unable to fetch userinfo");
  const json = await res.json();
  return BrukerInfoSchema.parse(json);
};

export const getCurrentAccessState = (
  dpal: DataProduktTilgangListe
): DataProduktTilgangListe => {
  if (!dpal) return [];
  let processedList: DataProduktTilgangListe = [];
  const reverseChronological = dpal.sort((x, y) =>
    x.time.localeCompare(y.time)
  );

  const deleteAllGrantsForSubject = (subject: string) => {
    if (!processedList) return;
    processedList.forEach((access, idx) => {
      if (subject === access.subject) {
        if (processedList) processedList.splice(idx, 1);
      }
    });
  };

  for (const dpa of reverseChronological) {
    switch (dpa.action) {
      case "grant":
        processedList.push({ ...dpa });
        break;
      case "delete":
        deleteAllGrantsForSubject(dpa.subject);
        break;
    }
  }

  return processedList;
};

export const isOwner = (produkt?: DataProdukt, teams?: string[]) => {
  if (!produkt || !teams) return false;
  if (produkt && teams.length) {
    return teams.includes(produkt.team);
  }
  return false;
};

export const deleteAccess = async (
  productID: string,
  subject: string,
  type: string
) => {
  const res = await fetch(`${API_ROOT}/access/${productID}`, {
    method: "delete",
    body: JSON.stringify({ subject, type }),
    credentials: "include",
  });

  if (!res.ok) throw new Error(await res.text());
};
