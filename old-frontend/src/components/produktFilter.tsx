import React, { useContext, useEffect, useState } from "react";
import "./produktFilter.less";
import Select from "react-select";
import { DataProduktListe } from "../lib/produktAPI";
import { Knapp } from "nav-frontend-knapper";
import { UserContext } from "../lib/userContext";

export const ProduktFilter: React.FC<{
  produkter?: DataProduktListe;
  filters: string[];
  setFilters: React.Dispatch<React.SetStateAction<string[]>>;
}> = ({ produkter, filters, setFilters }): JSX.Element => {
  const [options, setOptions] = useState<{ value: string; label: string }[]>(
    []
  );
  const user = useContext(UserContext);

  useEffect(() => {
    if (!produkter) return;
    let teams: string[] = [];

    produkter.forEach((p) => {
      if (!teams.includes(p.data_product.team)) teams.push(p.data_product.team);
    });

    setOptions(teams.map((t) => ({ value: t, label: t })));
  }, [produkter]);

  return (
    <div className={"produkt-filter"}>
      <label className={"skjemaelement__label"}>Filtrer på produkteier</label>
      <Select
        className={"filter-dropdown"}
        options={options}
        value={filters.map((t) => ({ value: t, label: t }))}
        onChange={(v) => setFilters(v.map((v) => v.value))}
        isMulti
      />
      {user?.teams && (
        <Knapp
          kompakt
          onClick={() => {
            setFilters(user.teams);
          }}
        >
          Vis mine
        </Knapp>
      )}
    </div>
  );
};

export default ProduktFilter;
