import React, { useContext, useEffect, useState } from "react";
import ReactMde from "react-mde";
import "react-mde/lib/styles/css/react-mde-all.css";
import ReactMarkdown from "react-markdown";

import {
  DataLager,
  DataLagerBigquery,
  DataLagerBigquerySchema,
  DataLagerBucket,
  DataLagerBucketSchema,
  DataProdukt,
  DataProduktSchema,
} from "../lib/produktAPI";
import { ZodError } from "zod";
import { Input, Select, SkjemaGruppe } from "nav-frontend-skjema";
import { UserContext } from "../lib/userContext";
import { Feilmelding } from "nav-frontend-typografi";
import { Fareknapp, Hovedknapp } from "nav-frontend-knapper";

const RessursVelger: React.FC<{
  datastore: DataLager | null;
  setDatastore: React.Dispatch<React.SetStateAction<DataLager | null>>;
  parseErrors: ZodError["formErrors"] | undefined;
}> = ({ datastore, setDatastore, parseErrors }) => {
  const bucketFields = (datastore: DataLagerBucket) => (
    <>
      <Input
        label="project_id"
        feil={parseErrors?.fieldErrors?.project_id}
        value={datastore.project_id || ""}
        onChange={(e) =>
          setDatastore({ ...datastore, project_id: e.target.value })
        }
      />
      <Input
        label="bucket_id"
        feil={parseErrors?.fieldErrors?.bucket_id}
        value={datastore.bucket_id || ""}
        onChange={(e) =>
          setDatastore({ ...datastore, bucket_id: e.target.value })
        }
      />
    </>
  );

  const bigqueryFields = (datastore: DataLagerBigquery) => (
    <>
      <Input
        label="Project ID"
        feil={parseErrors?.fieldErrors?.project_id}
        value={datastore.project_id || ""}
        onChange={(e) =>
          setDatastore({ ...datastore, project_id: e.target.value })
        }
      />
      <Input
        label="Resource ID"
        feil={parseErrors?.fieldErrors?.resource_id}
        value={datastore.resource_id || ""}
        onChange={(e) =>
          setDatastore({ ...datastore, resource_id: e.target.value })
        }
      />
      <Input
        label="Dataset ID"
        feil={parseErrors?.fieldErrors?.dataset_id}
        value={datastore.dataset_id || ""}
        onChange={(e) =>
          setDatastore({ ...datastore, dataset_id: e.target.value })
        }
      />
    </>
  );

  return (
    <>
      <Select
        feil={parseErrors?.fieldErrors?.type}
        label="Ressurstype"
        onChange={(e) => {
          if (e.target.value !== "")
            setDatastore({ type: e.target.value } as DataLager);
          else setDatastore(null);
        }}
      >
        <option key="" value="">
          Velg type
        </option>
        <option
          key="bigquery"
          selected={datastore?.type === "bigquery"}
          value="bigquery"
        >
          BigQuery
        </option>
        <option
          key="bucket"
          selected={datastore?.type === "bucket"}
          value="bucket"
        >
          Bucket
        </option>
      </Select>
      {datastore?.type === "bigquery" && bigqueryFields(datastore)}
      {datastore?.type === "bucket" && bucketFields(datastore)}
    </>
  );
};

export const ProduktSkjema: React.FC<{
  produkt?: DataProdukt;
  onProductReady: (dp: DataProdukt) => void;
  avbryt?: () => void;
}> = ({ produkt, onProductReady, avbryt }) => {
  const user = useContext(UserContext);

  const [navn, setNavn] = useState<string>("");
  const [beskrivelse, setBeskrivelse] = useState<string>("");
  const [eier, setEier] = useState<string>("");
  const [datastore, setDatastore] = useState<DataLager | null>(null);
  const [formErrors, setFormErrors] = useState<ZodError["formErrors"]>();
  const [dsFormErrors, setDsFormErrors] = useState<ZodError["formErrors"]>();
  const [markdownMode, setMarkdownMode] =
    useState<"write" | "preview">("write");

  // Correctly initialize form to default value on page render
  useEffect(() => setEier((e) => user?.teams?.[0] || e), [user]);

  useEffect(() => {
    if (!produkt) return;
    setNavn(produkt.name);
    setEier(produkt.team);
    setBeskrivelse(produkt.description);
    setDatastore(produkt?.datastore?.[0] || null);
  }, [produkt]);

  const parseDatastore = () => {
    if (datastore?.type === "bigquery") {
      DataLagerBigquerySchema.parse(datastore);
    } else if (datastore?.type === "bucket") {
      DataLagerBucketSchema.parse(datastore);
    } else {
      setDsFormErrors({
        formErrors: [],
        fieldErrors: { type: ["Required"] },
      });
    }
  };

  const handleSubmit = async (): Promise<void> => {
    // First make sure we have a valid datastore
    try {
      parseDatastore();
    } catch (e) {
      setDsFormErrors(e.flatten());
    }
    try {
      const nyttProdukt = DataProduktSchema.parse({
        name: navn,
        description: beskrivelse,
        datastore: [datastore],
        team: eier,
        access: {},
      });
      onProductReady(nyttProdukt);
    } catch (e) {
      console.log(e.toString());

      if (e instanceof ZodError) {
        setFormErrors(e.flatten());
      } else {
        setFormErrors({ formErrors: e.toString(), fieldErrors: {} });
      }
    }
  };
  return (
    <SkjemaGruppe>
      <Input
        label="Navn"
        value={navn}
        feil={formErrors?.fieldErrors?.name}
        onChange={(e) => setNavn(e.target.value)}
      />
      {formErrors?.fieldErrors?.description && (
        <Feilmelding>{formErrors?.fieldErrors?.description}</Feilmelding>
      )}
      <ReactMde
        selectedTab={markdownMode}
        onTabChange={setMarkdownMode}
        value={beskrivelse}
        onChange={setBeskrivelse}
        generateMarkdownPreview={(markdown) =>
          Promise.resolve(<ReactMarkdown children={markdown} />)
        }
      />
      <Select
        label="Eier (team)"
        selected={eier}
        feil={formErrors?.fieldErrors?.owner}
        onChange={(e) => setEier(e.target.value)}
        children={
          user?.teams
            ? user.teams.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))
            : null
        }
      />

      <RessursVelger
        datastore={datastore}
        setDatastore={setDatastore}
        parseErrors={dsFormErrors}
      />
      {!!formErrors?.formErrors?.length && (
        <Feilmelding>
          Feil: <br />
          <code>{formErrors.formErrors}</code>
        </Feilmelding>
      )}
      <div style={{ display: "flex", alignItems: "space-between" }}>
        {avbryt && <Fareknapp onClick={avbryt}>Avbryt</Fareknapp>}

        <Hovedknapp
          style={{ display: "block", marginLeft: "auto" }}
          onClick={async () => {
            await handleSubmit();
          }}
        >
          Submit
        </Hovedknapp>
      </div>
    </SkjemaGruppe>
  );
};

export default ProduktSkjema;
