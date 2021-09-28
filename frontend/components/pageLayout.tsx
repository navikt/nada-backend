import Header from "./header/header";
import styled from "styled-components";

const Container = styled.div`
    max-width: 80vw;
    margin: 0 auto;
`
const Main = styled.main`
    display: flex;
    min-width: 100%;
`
export const PageLayout = ({children}: { children: React.ReactNode }) => (
    <Container>
        <Header/>
        <Main>
            {children}
        </Main>
    </Container>
)

export default PageLayout
