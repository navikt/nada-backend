import styled from 'styled-components'
import Logo from './logo'
import User from "./user";

const HeaderBar = styled.div`
display: flex;
height: 100px;
justify-content: space-between;
`
export default function Header() {
    return (<HeaderBar><Logo/><User/></HeaderBar>)
}