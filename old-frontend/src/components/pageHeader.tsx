import React, { useContext } from "react";
import { BACKEND_ENDPOINT } from "../lib/produktAPI";
import { Hovedknapp } from "nav-frontend-knapper";
import { Child, Next } from "@navikt/ds-icons";
import { UserContext } from "../lib/userContext";
import { NaisPrideLogo } from "./svgIcons";
import { Link, useLocation, useHistory } from "react-router-dom";
import { Systemtittel } from "nav-frontend-typografi";
import "./pageHeader.less";

const BrukerBoks: React.FC = () => {
  const user = useContext(UserContext);
  const history = useHistory();
  const location = useLocation();

  const hasPendingRedirect = (): boolean =>
    !!window.localStorage.getItem("postLoginRedirect");
  const popPendingRedirect = (): string => {
    const newPath = window.localStorage.getItem("postLoginRedirect");
    window.localStorage.removeItem("postLoginRedirect");
    return newPath || "";
  };
  const pushPendingRedirect = (path: string) =>
    window.localStorage.setItem("postLoginRedirect", path);
  if (hasPendingRedirect()) history.push(popPendingRedirect());

  if (!user) {
    return (
      <a
        className="innloggingsknapp"
        onClick={() => {
          pushPendingRedirect(location.pathname);
        }}
        href={`${BACKEND_ENDPOINT}/login`}
      >
        <Hovedknapp className="innloggingsknapp">logg inn</Hovedknapp>
      </a>
    );
  } else {
    return (
      <div className={"brukerboks"}>
        <Child />
        {user.email}
      </div>
    );
  }
};

export const PageHeader: React.FC<{ crumbs: string | null }> = ({ crumbs }) => {
  return (
    <header>
      <NaisPrideLogo />
      <div className={"crumb-trail"}>
        <Link to="/">
          <Systemtittel>Dataprodukter</Systemtittel>
        </Link>
        {crumbs ? <Next className="pil" /> : null}
        {crumbs ? <Systemtittel>{crumbs}</Systemtittel> : null}
      </div>
      <BrukerBoks />
    </header>
  );
};

export default PageHeader;
