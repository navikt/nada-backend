import React from "react";
import "nav-frontend-tabell-style";
import NavFrontendSpinner from "nav-frontend-spinner";
import { DataProduktListe, DataProduktResponse } from "../lib/produktAPI";
import { Link } from "react-router-dom";
import "./produktTabell.less";
interface ProduktProps {
  produkt: DataProduktResponse;
}

const Produkt = ({ produkt }: ProduktProps) => {
  if (!produkt.data_product) return null;
  return (
    <tr>
      <td>{produkt.data_product.team}</td>
      <td>
        <Link to={`/produkt/${produkt.id}`}>{produkt.data_product.name}</Link>
      </td>
      <td>
        {produkt.data_product.datastore &&
          produkt.data_product.datastore[0].type}
      </td>
    </tr>
  );
};

export const ProduktTabell: React.FC<{ produkter?: DataProduktListe }> = ({
  produkter,
}) => {
  return (
    <div className="produkt-liste">
      <table className="tabell">
        <thead>
          <tr>
            <th>Produkteier</th>
            <th>Navn</th>
            <th>Type</th>
          </tr>
        </thead>
        <tbody>
          {produkter &&
            produkter.map((x) => <Produkt key={x.id} produkt={x} />)}
        </tbody>
      </table>
      {typeof produkter !== "undefined" && !produkter.length ? (
        <p style={{ textAlign: "center", fontStyle: "italic", margin: "2%" }}>
          Ingen dataprodukter i katalogen
        </p>
      ) : null}
      {!produkter && (
        <p style={{ textAlign: "center", margin: "2%" }}>
          <NavFrontendSpinner />
        </p>
      )}
    </div>
  );
};

export default ProduktTabell;
