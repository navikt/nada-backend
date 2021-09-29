import SearchResult from './searchresult'
import styled from "styled-components";
import {navGra60} from "../../styles/constants";
import {SearchBoxProps} from "../search/search";

const ResultsBox = styled.div`
    border: 2px solid ${navGra60};
    flex-grow: 1;
    padding: 15px;
`
export interface ResultProps{
    data: []
    error: string
}

export function Results({ data, error }: ResultProps) {
    console.log(data)
    return (
        <ResultsBox>
            <SearchResult result={{name: 'foo', description: 'bar'}}/>
            <SearchResult result={{name: 'foo', description: 'bar'}}/>
            <SearchResult result={{name: 'foo', description: 'bar'}}/>
            <SearchResult result={{name: 'foo', description: 'bar'}}/>
        </ResultsBox>
    )
}

export default Results