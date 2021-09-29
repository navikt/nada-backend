import SearchResult from './searchresult'
import styled from "styled-components";
import {navGra60} from "../../styles/constants";
import {SearchBoxProps} from "../search/search";
import {Loader} from "@navikt/ds-react";
import {SearchResultEntry} from "../../lib/schema_types";


const ResultsBox = styled.div`
    border: 2px solid ${navGra60};
    flex-grow: 1;
    padding: 15px;
`

export interface ResultProps {
    data: SearchResultEntry[]
    error: string
}

export function Results({data, error}: ResultProps) {
    if (error) {
        return (
            <div>error</div>
        )
    }
    if (!data) {
        return (<div>  <Loader transparent /></div>)
    }

    return (
        <ResultsBox>
            {data.map((d) => {return (<SearchResult key={d.id} searchResultEntry={d}/>)})}
        </ResultsBox>
    )
}

export default Results