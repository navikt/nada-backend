import styled from "styled-components";
import Searchfield from './searchfield'


const SearchBox = styled.div`
    border: 2px solid black;
    width: 25%;
    padding: 15px;
    margin-right: 25px;
`

export default function Search(){
    return (<SearchBox><Searchfield/></SearchBox>)
}