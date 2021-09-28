import styled from 'styled-components'
import Logo from './logo'
import User from "./user";

const HeaderBar = styled.header`
    display: flex;
    margin: 40px 0;
    justify-content: space-between;
`
export default function Header() {
    return (<HeaderBar><Logo/><User/></HeaderBar>)
}