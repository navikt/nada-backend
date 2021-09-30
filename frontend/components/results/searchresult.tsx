import styled from "styled-components";
import Link from 'next/link'
import { navGra20, navGra40 } from "../../styles/constants";
import { components } from "../../lib/schema"

const SearchResultDiv = styled.div`
    background-color: ${navGra20};
    padding: 5px;
    margin-bottom: 5px;
    h1 {
        font-size: 1.5em;
        margin: 0;
    }
    cursor: pointer;
    :hover {
        background-color: ${navGra40};
    }
`


type SearchResultEntry = components["schemas"]["SearchResultEntry"]

export interface SearchResultProps {
    searchResultEntry: SearchResultEntry
}

export const SearchResult = ({ searchResultEntry }: SearchResultProps) => {

    if (searchResultEntry.type === "dataproduct") {
        return (
            <SearchResultDiv>
                <Link href={`/dataproduct/${searchResultEntry.id}`}>
                    <h1>{searchResultEntry.name}</h1>
                </Link>
                <p>{searchResultEntry.excerpt}</p>
            </SearchResultDiv>
        )
    } else if (searchResultEntry.type === "datapackage") {
        return (
            <SearchResultDiv>
                <Link href={`/datapackage/${searchResultEntry.id}`}>
                    <h1>{searchResultEntry.name}</h1>
                </Link>
                <p>{searchResultEntry.excerpt}</p>
            </SearchResultDiv>
        )
    }
    
    return (
        <SearchResultDiv>
            <Link href={`/dataset/${searchResultEntry.id}`}>
                <h1>{searchResultEntry.name}</h1>
            </Link>
            <p>{searchResultEntry.excerpt}</p>
        </SearchResultDiv>
    )
}

export default SearchResult