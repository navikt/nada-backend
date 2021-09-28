import styled from 'styled-components'
import type {NextPage} from 'next'
import Header from '../components/header/header'
import Search from "../components/search/search";
import Results from "../components/results/results";

const Container = styled.div`
`
const Main = styled.div`
display: flex;
`

const Home: NextPage = () => {
    return (
        <Container>
            <Header/>
            <Main>
                <Search/>
                <Results/>
            </Main>
        </Container>
    )
}

export default Home
