import React from "react";
import {
  DataLager,
  DataLagerBigquery,
  DataLagerBucket,
  DataProduktResponse,
} from "../lib/produktAPI";
import { BigQueryIcon, BucketIcon } from "./svgIcons";
import "./produktDatalager.less";
import { Normaltekst, Undertittel } from "nav-frontend-typografi";

export const DatalagerInfo: React.FC<{ ds: DataLager }> = ({ ds }) => {
  const BigQueryEntry = (e: DataLagerBigquery) => (
    <div className={"bigqueryentry datalagerentry"}>
      <BigQueryIcon />
      <div>
        <Undertittel>BigQuery</Undertittel>
        <ul>
          <li>
            Dataset ID: <em>{e.dataset_id}</em>
          </li>
          <li>
            Project ID: <em>{e.project_id}</em>
          </li>
          <li>
            Resource ID: <em>{e.resource_id}</em>
          </li>
        </ul>
      </div>
    </div>
  );
  const BucketEntry = (e: DataLagerBucket) => (
    <div className={"bucketentry datalagerentry"}>
      <BucketIcon />
      <div>
        <Undertittel>Bucket</Undertittel>
        <ul>
          <li>
            Project ID: <em>{e.project_id}</em>
          </li>
          <li>
            Bucket ID: <em>{e.bucket_id}</em>
          </li>
        </ul>
      </div>
    </div>
  );

  return (
    <>
      {ds.type === "bigquery" && BigQueryEntry(ds)}
      {ds.type === "bucket" && BucketEntry(ds)}
    </>
  );
};

export const ProduktDatalager: React.FC<{ produkt: DataProduktResponse }> = ({
  produkt,
}) => (
  <>
    {produkt.data_product?.datastore ? (
      produkt.data_product?.datastore.map((ds, index) => (
        <DatalagerInfo key={index} ds={ds} />
      ))
    ) : (
      <div>
        <Normaltekst>Ingen datalagre definert</Normaltekst>
      </div>
    )}
  </>
);
