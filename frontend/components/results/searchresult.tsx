import styled from "styled-components";
import {NavLysGra} from "../../styles/constants";

const SearchResultDiv = styled.div`
    background-color: ${NavLysGra};
    padding: 5px;
    margin-bottom: 5px;
    h1 {
        font-size: 1.5em;
        margin: 0;
    }
`

export type DataProduct = {
    name: string,
    description: string,
}

export interface SearchResultProps {
    result: DataProduct // | DataPackage | DataSet
}

export const SearchResult = ({result}: SearchResultProps) => (
    <SearchResultDiv>
        <h1>{result.name}</h1>
        <p>{result.description}</p>
    </SearchResultDiv>
)

export default SearchResult