import React, { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { GiTilgang, SlettProdukt } from "../components/produktTilgangModaler";
import { Feilmelding, Sidetittel, Systemtittel } from "nav-frontend-typografi";
import {
  DataProduktResponse,
  DataProduktTilgangListe,
  hentProdukt,
  hentTilganger,
} from "../lib/produktAPI";
import NavFrontendSpinner from "nav-frontend-spinner";
import "./produktDetalj.less";
import { ProduktFaktaboks } from "../components/produktDetaljFaktaboks";
import { ProduktKnapperad } from "../components/produktDetaljKnapperad";
import { ProduktDatalager } from "../components/produktDatalager";
import { ProduktTilganger } from "../components/produktDetaljTilganger";

interface ProduktDetaljParams {
  produktID: string;
}

const FaktaboksAvsnitt: React.FC<{}> = ({ children }) => (
  <div className={"faktaboks-avsnitt"}>{children}</div>
);

export const ProduktDetalj: React.FC<{
  setCrumb: React.Dispatch<React.SetStateAction<string | null>>;
}> = ({ setCrumb }): JSX.Element => {
  let { produktID } = useParams<ProduktDetaljParams>();
  const [tilgangIsOpen, setTilgangIsOpen] = useState<boolean>(false);
  const [isOpen, setIsOpen] = useState<boolean>(false);

  const [produkt, setProdukt] = useState<DataProduktResponse | null>(null);
  const [error, setError] = useState<string | null>();
  const [tilganger, setTilganger] =
    useState<DataProduktTilgangListe | null>(null);
  const [tilgangerError, setTilgangerError] = useState<string | null>();

  useEffect(() => {
    if (!produkt) return;
    hentTilganger(produkt.id)
      .then((p) => {
        setTilganger(p);
        setTilgangerError(null);
      })
      .catch((e) => {
        console.log(e.toString());
        setTilgangerError(e.toString());
      });
  }, [produkt]);

  useEffect(() => {
    hentProdukt(produktID)
      .then((p) => {
        setProdukt(p);
        setError(null);
      })
      .catch((e) => {
        setError(e.toString());
      });
  }, [produktID]);

  const refreshAccessState = () => {
    setTilgangIsOpen(false);
    // TODO: Use hooks more elegantly
    hentProdukt(produktID)
      .then((p) => {
        setProdukt(p);
        setError(null);
      })
      .catch((e) => {
        setError(e.toString());
      });
  };

  useEffect(() => {
    if (produkt != null) {
      setCrumb(produkt?.data_product.name || null);
    }
  }, [produkt, setCrumb]);

  if (error || tilgangerError)
    return (
      <Feilmelding>
        <code>{error}</code>
        <code>{tilgangerError}</code>
      </Feilmelding>
    );

  if (typeof produkt == "undefined")
    return (
      <div style={{ textAlign: "center" }}>
        <NavFrontendSpinner />
      </div>
    );

  if (produkt == null) return <></>;

  return (
    <div className="produkt-detalj">
      <div className={"faktaboks"}>
        <SlettProdukt
          isOpen={isOpen}
          setIsOpen={setIsOpen}
          produktID={produkt.id}
        />

        <GiTilgang
          tilgangIsOpen={tilgangIsOpen}
          refreshAccessState={refreshAccessState}
          produkt={produkt}
        />
        <Sidetittel>
          <Link
            to={{
              pathname: "/",
              search: `?teams=${produkt.data_product.team}`,
            }}
          >
            {produkt.data_product.team}
          </Link>{" "}
          / {produkt.data_product.name}
        </Sidetittel>

        <FaktaboksAvsnitt>
          <Systemtittel>Produkt</Systemtittel>

          <ProduktFaktaboks tilganger={tilganger} produkt={produkt} />
        </FaktaboksAvsnitt>
        <FaktaboksAvsnitt>
          <Systemtittel>Datalager</Systemtittel>
          <ProduktDatalager produkt={produkt} />
        </FaktaboksAvsnitt>
        <FaktaboksAvsnitt>
          <Systemtittel>Tilganger</Systemtittel>
          <ProduktTilganger
            produkt={produkt}
            tilganger={tilganger}
            refreshAccessState={refreshAccessState}
          />
        </FaktaboksAvsnitt>
      </div>

      <ProduktKnapperad
        produkt={produkt}
        tilganger={tilganger}
        openSlett={() => setIsOpen(true)}
        openTilgang={() => setTilgangIsOpen(true)}
      />
    </div>
  );
};

export default ProduktDetalj;
