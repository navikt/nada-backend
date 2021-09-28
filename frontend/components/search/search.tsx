import styled from "styled-components";
import {TextField} from "@navikt/ds-react";

const SearchDiv = styled.div`
    border: 2px solid black;
    width: 25%;
    padding: 15px;
    margin-right: 25px;
`
export interface SearchBoxProps{
    query: string
    setQuery: React.Dispatch<React.SetStateAction<string>>
}

export default function SearchBox({query,setQuery}: SearchBoxProps){
    return (<SearchDiv><TextField label={""}/></SearchDiv>)
}