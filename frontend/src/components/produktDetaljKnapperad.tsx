import { Fareknapp, Knapp } from "nav-frontend-knapper";
import React, { useContext } from "react";
import { UserContext } from "../lib/userContext";
import {
  DataProduktResponse,
  DataProduktTilgangListe,
  getCurrentAccessState,
  isOwner,
} from "../lib/produktAPI";
import { useHistory } from "react-router-dom";

export const ProduktKnapperad: React.FC<{
  produkt: DataProduktResponse;
  tilganger: DataProduktTilgangListe;
  openSlett: () => void;
  openTilgang: () => void;
}> = ({ produkt, tilganger, openSlett, openTilgang }) => {
  const history = useHistory();

  const userContext = useContext(UserContext);
  const harTilgang = (tilganger: DataProduktTilgangListe): boolean => {
    if (!tilganger) return false;
    const tilgangerBehandlet = getCurrentAccessState(tilganger);
    if (!tilgangerBehandlet) return false;

    for (const tilgang of tilgangerBehandlet) {
      if (tilgang.subject === userContext?.email) {
        if (tilgang?.expires && new Date(tilgang.expires) > new Date()) {
          return true;
        }
      }
    }

    return false;
  };

  const ownsProduct = () => isOwner(produkt?.data_product, userContext?.teams);
  return (
    <div className="knapperad">
      {ownsProduct() && (
        <>
          <Fareknapp onClick={() => openSlett()}>Slett</Fareknapp>
          <Knapp
            onClick={() => {
              history.push(`${produkt.id}/rediger`);
            }}
          >
            Endre produkt
          </Knapp>
        </>
      )}

      {userContext &&
        !harTilgang(tilganger) &&
        produkt?.data_product.datastore && (
          <Knapp onClick={() => openTilgang()}>Få tilgang</Knapp>
        )}
    </div>
  );
};
