import type { NextPage } from "next"
import Search from "../components/search/search"
import Results from "../components/results/results"
import { Layout } from "../components/layout"

const Home: NextPage = () => {
  return (
    <Layout>
      <Search />
      <Results />
    </Layout>
  )
}

export default Home
