import React, { useContext, useEffect, useState } from "react";
import ProduktTabell from "../components/produktTabell";
import { DataProduktListe, hentProdukter } from "../lib/produktAPI";
import ProduktFilter from "../components/produktFilter";
import { Add } from "@navikt/ds-icons";
import { Link } from "react-router-dom";
import { UserContext } from "../lib/userContext";
import { useLocation } from "react-router-dom";
import { useHistory } from "react-router-dom";

import NavFrontendSpinner from "nav-frontend-spinner";
import { Knapp } from "nav-frontend-knapper";
import "./produktListe.less";

const ProduktNyKnapp = (): JSX.Element => (
  <div className={"nytt-produkt"}>
    <Link to="/produkt/nytt">
      <Knapp>
        <Add />
        Nytt produkt
      </Knapp>
    </Link>
  </div>
);

function useQuery() {
  return new URLSearchParams(useLocation().search);
}

export const ProduktListe = (): JSX.Element => {
  const user = useContext(UserContext);
  const query = useQuery();
  const history = useHistory();

  const queryParameters = (query.get("teams") || null)?.split(",");

  const [error, setError] = useState<string | null>();
  const [filters, setFilters] = useState<string[]>(
    queryParameters ? queryParameters : []
  );
  const [produkter, setProdukter] = useState<DataProduktListe>();

  useEffect(() => {
    if (!queryParameters) {
      const localStorageFilters = window.localStorage.getItem("filters");
      if (localStorageFilters?.length)
        setFilters(JSON.parse(localStorageFilters));
    }
  }, [queryParameters]);

  useEffect(() => {
    window.localStorage.setItem("filters", JSON.stringify(filters));

    history.push({
      search: filters.length ? "?teams=" + filters.join(",") : "",
    });
  }, [filters, history]);

  useEffect(() => {
    hentProdukter()
      .then((produkter) => {
        setProdukter(produkter);
        setError(null);
      })
      .catch((e) => {
        console.log(e);
        setError(e.toString());
      });
  }, []);

  if (error) {
    setTimeout(() => window.location.reload(), 1500);
    return (
      <div className={"feilBoks"}>
        <div>
          <h1>Kunne ikke hente produkter</h1>
        </div>
        <div>
          <h2>
            <NavFrontendSpinner /> Prøver på nytt...
          </h2>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="filter-and-button">
        <ProduktFilter
          produkter={produkter}
          filters={filters}
          setFilters={setFilters}
        />
        {user && <ProduktNyKnapp />}
      </div>
      <ProduktTabell
        produkter={produkter?.filter(
          (p) => !filters.length || filters.includes(p.data_product.team)
        )}
      />
    </div>
  );
};
