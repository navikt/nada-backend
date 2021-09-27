import React, { useState, useEffect } from "react";
import ReactDOM from "react-dom";
import "./index.less";
import reportWebVitals from "./lib/reportWebVitals";
import { hentBrukerInfo, BrukerInfo } from "./lib/produktAPI";
import { ProduktListe } from "./pages/produktListe";
import {
  useHistory,
  BrowserRouter as Router,
  Switch,
  Route,
} from "react-router-dom";
import ProduktDetalj from "./pages/produktDetalj";
import ProduktNytt from "./pages/produktNytt";
import { Systemtittel } from "nav-frontend-typografi";

import { UserContext } from "./lib/userContext";
import PageHeader from "./components/pageHeader";
import ProduktOppdatering from "./pages/produktOppdatering";

const App = (): JSX.Element => {
  const [crumb, setCrumb] = useState<string | null>(null);
  const [user, setUser] = useState<BrukerInfo | null>(null);
  const history = useHistory();

  const validateSession = (): void => {
    hentBrukerInfo()
      .then((bruker) => {
        setUser(bruker);
      })
      .catch(() => {
        setUser(null);
      });
  };

  useEffect(() => {
    validateSession();
    history.listen(validateSession);
  }, [history]);

  return (
    <div className={"dashboard-main app"}>
      <UserContext.Provider value={user}>
        <PageHeader crumbs={crumb} />
        <main>
          <Switch>
            <Route
              path="/produkt/nytt"
              children={() => {
                setCrumb("Nytt produkt");
                return <ProduktNytt />;
              }}
            />
            <Route
              path="/produkt/:produktID/rediger"
              children={<ProduktOppdatering />}
            />
            <Route
              path="/produkt/:produktID"
              children={<ProduktDetalj setCrumb={setCrumb} />}
            />
            <Route
              exact
              path="/"
              children={() => {
                setCrumb(null);
                return <ProduktListe />;
              }}
            />
            <Route path="*">
              <Systemtittel>404 - ikke funnet</Systemtittel>
            </Route>
          </Switch>
        </main>
      </UserContext.Provider>
    </div>
  );
};

ReactDOM.render(
  <React.StrictMode>
    <Router>
      <App />
    </Router>
  </React.StrictMode>,
  document.getElementById("root")
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
