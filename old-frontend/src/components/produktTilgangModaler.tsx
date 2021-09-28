import { Feilmelding, Systemtittel } from "nav-frontend-typografi";
import React, { useState, useContext } from "react";
import {
  giTilgang,
  DataProduktResponse,
  slettProdukt,
} from "../lib/produktAPI";
import "react-datepicker/dist/react-datepicker.css";
import { UserContext } from "../lib/userContext";
import Modal from "nav-frontend-modal";
import { ToggleKnappPure } from "nav-frontend-toggle";
import { useHistory } from "react-router-dom";
import { Fareknapp, Hovedknapp } from "nav-frontend-knapper";

import DatePicker from "react-datepicker";
import "./produktTilgangModaler.less";
import { Child, Warning } from "@navikt/ds-icons";

interface SlettProduktProps {
  produktID: string;
  isOpen: boolean;
  setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
}

export const GiTilgang: React.FC<{
  produkt: DataProduktResponse;
  tilgangIsOpen: boolean;
  refreshAccessState: () => void;
}> = ({ produkt, tilgangIsOpen, refreshAccessState }) => {
  const [endDate, setEndDate] = useState<Date | null>(new Date());
  const [evig, setEvig] = useState<boolean>(false);
  const [feilmelding, setFeilmelding] = useState<string | null>(null);
  const userContext = useContext(UserContext);
  if (!userContext) return null;

  const handleSubmit = async () => {
    try {
      await giTilgang(
        produkt,
        userContext.email,
        evig ? null : endDate?.toISOString() || null
      );
      setFeilmelding(null);
      refreshAccessState();
    } catch (e) {
      setFeilmelding(e.toString());
    }
  };

  console.log(evig);

  return (
    <Modal
      appElement={document.getElementById("app") || undefined}
      isOpen={tilgangIsOpen}
      onRequestClose={() => refreshAccessState()}
      closeButton={false}
      contentLabel="Gi tilgang"
      className={"gitilgang"}
    >
      <Systemtittel>Gi tilgang</Systemtittel>
      {feilmelding ? <Feilmelding>{feilmelding}</Feilmelding> : null}
      <div className={"tilgang-skjema"}>
        <div className={"tilgang-rad"}>
          <label>Til:</label>
          <div className={"brukerboks"}>
            <Child />
            {userContext.email}
          </div>
        </div>
        <div className={"tilgang-rad"}>
          <label>Sluttdato:</label>
          <div className={evig ? "datovalg datovalg-disabled" : "datovalg"}>
            <div className={"kalender"} onClick={() => setEvig(false)}>
              <DatePicker
                selected={endDate}
                onChange={(e) => setEndDate(e as Date)}
                selectsEnd
                endDate={endDate}
                startDate={new Date()}
                minDate={new Date()}
                inline
              />
            </div>

            <ToggleKnappPure pressed={evig} onClick={(ev) => setEvig(!evig)}>
              <Warning style={{ marginRight: "0.25rem" }} />
              Evig
            </ToggleKnappPure>
          </div>
        </div>
      </div>
      <div className={"knapperad"}>
        <Fareknapp onClick={() => refreshAccessState()}>Avbryt</Fareknapp>
        <Hovedknapp className={"bekreft"} onClick={() => handleSubmit()}>
          Bekreft
        </Hovedknapp>
      </div>
    </Modal>
  );
};

export const SlettProdukt = ({
  produktID,
  isOpen,
  setIsOpen,
}: SlettProduktProps): JSX.Element => {
  const [error, setError] = useState<string | null>(null);
  const history = useHistory();

  const deleteProduct = async (id: string) => {
    try {
      await slettProdukt(id);
      history.push("/");
    } catch (e) {
      setError(e.toString());
    }
  };

  return (
    <Modal
      appElement={document.getElementById("app") || undefined}
      isOpen={isOpen}
      onRequestClose={() => setIsOpen(false)}
      closeButton={true}
      contentLabel="Min modalrute"
    >
      <div className="slette-bekreftelse">
        <Systemtittel>Er du sikker?</Systemtittel>
        {error ? <p>{error}</p> : null}
        <Fareknapp onClick={() => deleteProduct(produktID)}>
          {error ? "Prøv igjen" : "Ja"}
        </Fareknapp>
      </div>
    </Modal>
  );
};
