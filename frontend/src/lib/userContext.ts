import { createContext } from "react";
import { BrukerInfo } from "./produktAPI";

export const UserContext = createContext<BrukerInfo | null>(null);
