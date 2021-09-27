import React from "react";
import {
  DataProduktResponse,
  DataProduktTilgangListe,
} from "../lib/produktAPI";
import moment from "moment";
import { Ingress, Normaltekst } from "nav-frontend-typografi";
import ReactMarkdown from "react-markdown";

export const ProduktFaktaboks: React.FC<{
  produkt: DataProduktResponse;
  tilganger: DataProduktTilgangListe;
}> = ({ tilganger, produkt }) => {
  moment.locale("nb");

  return (
    <div className={"infoBoks"}>
      <Ingress className="beskrivelse">
        {(produkt.data_product?.description && (
          <ReactMarkdown children={produkt.data_product?.description} />
        )) ||
          "Ingen beskrivelse"}
      </Ingress>
      <Normaltekst>
        Opprettet: {moment(produkt.created).format("LLL")}
      </Normaltekst>

      {produkt.created !== produkt.updated ? (
        <Normaltekst>
          Oppdatert: {moment(produkt.updated).fromNow()}
        </Normaltekst>
      ) : null}
    </div>
  );
};
